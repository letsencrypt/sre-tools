package reviews

import (
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	rotationLocation = "US/Pacific"
	mdspURL          = "https://groups.google.com/a/mozilla.org/forum/feed/dev-security-policy/topics/rss_v2_0.xml"
	mzCAURL          = "https://bugzilla.mozilla.org/buglist.cgi?bug_status=__open__&chfield=%5BBug%20creation%5D&chfieldfrom=YYYY-MM-DD&chfieldto=YYYY-MM-DD&component=CA%20Certificate%20Compliance&product=NSS&query_format=advanced&title=Bug%20List&query_based_on=&columnlist=short_desc%2Copendate&ctype=csv&human=1"
	mzLEURL          = "https://bugzilla.mozilla.org/buglist.cgi?bug_status=UNCONFIRMED&bug_status=NEW&bug_status=ASSIGNED&bug_status=REOPENED&bug_status=VERIFIED&list_id=15753483&component=CA%20Certificate%20Compliance&component=CA%20Certificate%20Root%20Program&short_desc_type=allwordssubstr&query_format=advanced&short_desc=Let%27s%20Encrypt&query_based_on=&columnlist=short_desc%2Copendate&ctype=csv&human=1"
	userAgent        = "Firefox: Mozilla/5.0 (Windows NT 6.3; WOW64; rv:41.0) Gecko/20100101 Firefox/41.0"
	tsPlaceholder    = "YYYY-MM-DD"
)

type reviewEntry struct {
	title   string
	link    string
	created time.Time
}

func (r reviewEntry) String() string {
	return fmt.Sprintf("- [ ] [%s](%s)\n", r.title, r.link)
}

func getRotationLocation() (*time.Location, error) {
	return time.LoadLocation(rotationLocation)
}

// getRotationPeriod returns the begin and end timestamps for Monday at 12:00PM
// in the same timezone as the compliance rotation.
func getRotationPeriod() (time.Time, time.Time, error) {
	loc, err := getRotationLocation()
	if err != nil {
		return time.Time{}, time.Time{}, nil
	}

	now := time.Now().In(loc)
	end := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	var begin time.Time
	for {
		if end.Weekday() == 1 {
			begin = end.Add(-(24 * time.Hour) * 7)
			break
		} else {
			end = end.Add(-(24 * time.Hour))
		}
	}
	return begin, end, nil
}

func requestFeedContents(feedURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return nil, err
	}

	// TODO(@beautifulentropy): We should contact mozdev admins and have a
	// custom user-agent added to their bot list. For now we're faking Firefox
	// useragent to avoid the BugZilla Bot hammer. We're not a bot and this tool
	// will not call the API more than once or twice per week.
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

type mdspFeed struct {
	Items []mdspEntry `xml:"channel>item"`
}

type mdspEntry struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
}

// created parses the timestamp and returns it for the same timezone as the
// compliance rotation.
func (m mdspEntry) created() (time.Time, error) {
	pubTime, err := time.Parse(time.RFC1123, m.PubDate)
	if err != nil {
		return time.Time{}, err
	}

	loc, err := getRotationLocation()
	if err != nil {
		return time.Time{}, err
	}
	return pubTime.In(loc), nil
}

func (m mdspEntry) makeReviewEntry() (reviewEntry, error) {
	created, err := m.created()
	if err != nil {
		return reviewEntry{}, err
	}
	return reviewEntry{m.Title, m.Link, created}, nil
}

func parseMDSPTopics(contents []byte) ([]mdspEntry, error) {
	var feed mdspFeed
	err := xml.Unmarshal(contents, &feed)
	if err != nil {
		return nil, err
	}
	return feed.Items, nil
}

func getMDSPTopics() ([]mdspEntry, error) {
	contents, err := requestFeedContents(mdspURL)
	if err != nil {
		return nil, err
	}

	topics, err := parseMDSPTopics(contents)
	if err != nil {
		return nil, err
	}
	return topics, nil
}

func GetMDSPReviewsForWeek() ([]reviewEntry, error) {
	topics, err := getMDSPTopics()
	if err != nil {
		return nil, err
	}

	begin, end, err := getRotationPeriod()
	if err != nil {
		return nil, err
	}

	var weekTopics []reviewEntry
	for _, topic := range topics {
		review, err := topic.makeReviewEntry()
		if err != nil {
			return nil, err
		}

		if review.created.Sub(begin) > 0 && review.created.Sub(end) <= 0 {
			weekTopics = append(weekTopics, review)
		}
	}
	return weekTopics, nil
}

type bzBugEntry struct {
	id      string
	summary string
	opened  string
}

// created parses the timestamp and returns it for the same timezone as the
// compliance rotation.
func (m bzBugEntry) created() (time.Time, error) {
	pubTime, err := time.Parse("2006-01-02 15:04:05 MST", m.opened)
	if err != nil {
		return time.Time{}, err
	}

	loc, err := getRotationLocation()
	if err != nil {
		return time.Time{}, err
	}
	return pubTime.In(loc), nil
}

func (m bzBugEntry) makeReviewEntry() (reviewEntry, error) {
	created, err := m.created()
	if err != nil {
		return reviewEntry{}, err
	}
	return reviewEntry{
		fmt.Sprintf("**%s** %s", m.id, m.summary),
		fmt.Sprintf("https://bugzilla.mozilla.org/show_bug.cgi?id=%s", m.id),
		created,
	}, nil
}

func parseBZBugs(contents []byte) ([]bzBugEntry, error) {
	zone, _ := time.Now().Zone()
	csvData := bytes.NewReader(contents)
	csvReader := csv.NewReader(csvData)

	var bugs []bzBugEntry
	for {
		record, err := csvReader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		// TZ is not provided but it will be the same as the caller.
		opened := record[2] + " " + zone
		bugs = append(bugs, bzBugEntry{record[0], record[1], opened})
	}

	// "< 2" because the CSV header will always be returned.
	if len(bugs) < 2 {
		fmt.Println("0 bugzilla bugs were found")
	}

	// Return [1:] to skip the header.
	return bugs[1:], nil
}

func makeBZURL(url string) (string, error) {
	begin, end, err := getRotationPeriod()
	if err != nil {
		return "", err
	}
	return strings.Replace(
		strings.Replace(
			url, tsPlaceholder, begin.Format("2006-01-02"), 1,
		), tsPlaceholder, end.Format("2006-01-02"), 1,
	), nil
}

func getMozBugs(url string) ([]bzBugEntry, error) {
	url, err := makeBZURL(url)
	if err != nil {
		return nil, err
	}

	contents, err := requestFeedContents(url)
	if err != nil {
		return nil, err
	}

	bugs, err := parseBZBugs(contents)
	if err != nil {
		return nil, err
	}
	return bugs, nil
}

func GetMozDevCAReviewsForWeek() ([]reviewEntry, error) {
	bugs, err := getMozBugs(mzCAURL)
	if err != nil {
		return nil, err
	}

	begin, end, err := getRotationPeriod()
	if err != nil {
		return nil, err
	}

	var weekBugs []reviewEntry
	for _, bug := range bugs {
		review, err := bug.makeReviewEntry()
		if err != nil {
			return nil, err
		}

		if review.created.Sub(begin) > 0 && review.created.Sub(end) <= 0 {
			weekBugs = append(weekBugs, review)
		}
	}
	return weekBugs, nil
}

func GetMozLEReviewsForWeek() ([]reviewEntry, error) {
	bugs, err := getMozBugs(mzLEURL)
	if err != nil {
		return nil, err
	}

	var weekBugs []reviewEntry
	for _, bug := range bugs {
		review, err := bug.makeReviewEntry()
		if err != nil {
			return nil, err
		}
		weekBugs = append(weekBugs, review)
	}
	return weekBugs, nil
}
