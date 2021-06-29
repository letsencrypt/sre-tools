package reviews

import (
	"reflect"
	"testing"
)

const mdspXML = `
<rss version="2.0">
<channel>
<item>
<title>1</title>
<link>http://a.link.to</link>
<pubDate>Thu, 24 Jun 2021 16:38:19 UTC</pubDate>
</item>
<item>
<title>2</title>
<link>http://a.link.to</link>
<pubDate>Wed, 23 Jun 2021 10:33:32 UTC</pubDate>
</item>
<item>
<title>3</title>
<link>http://a.link.to</link>
<pubDate>Sat, 19 Jun 2021 18:32:53 UTC</pubDate>
</item>
</channel>
</rss>
`

func Test_parseMDSPTopics(t *testing.T) {
	mdspBytes := []byte(mdspXML)
	mdspWant := []mdspEntry{
		{"1", "http://a.link.to", "Thu, 24 Jun 2021 16:38:19 UTC"},
		{"2", "http://a.link.to", "Wed, 23 Jun 2021 10:33:32 UTC"},
		{"3", "http://a.link.to", "Sat, 19 Jun 2021 18:32:53 UTC"},
	}

	type args struct {
		contents []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []mdspEntry
		wantErr bool
	}{
		{"happy path", args{mdspBytes}, mdspWant, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMDSPTopics(tt.args.contents)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMDSPTopics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

const bzCSV = `"Bug ID","Summary","Opened"
1341,"5","2021-06-28 10:57:01"
1337,"1","2021-06-23 02:20:38"
1338,"2","2021-06-23 02:24:55"
1339,"3","2021-06-23 02:28:31"
1340,"4","2021-06-28 10:20:19"`

func Test_parseBZBugs(t *testing.T) {
	bzBugBytes := []byte(bzCSV)
	bzBugWant := []bzBugEntry{
		{"1341", "5", "2021-06-28 10:57:01 PDT"},
		{"1337", "1", "2021-06-23 02:20:38 PDT"},
		{"1338", "2", "2021-06-23 02:24:55 PDT"},
		{"1339", "3", "2021-06-23 02:28:31 PDT"},
		{"1340", "4", "2021-06-28 10:20:19 PDT"},
	}

	type args struct {
		contents []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []bzBugEntry
		wantErr bool
	}{
		{"Happy path", args{bzBugBytes}, bzBugWant, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBZBugs(tt.args.contents)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBZBugs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseBZBugs() = %v, want %v", got, tt.want)
			}
		})
	}
}
