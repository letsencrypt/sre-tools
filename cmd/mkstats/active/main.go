package main

import (
	"encoding/csv"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

type data struct {
	date         string
	serialBytes  []byte
	reversedName string
}

const dateFormat = "2006-01-02"
const dateFormatFull = "2006-01-02 15:04:05"

func ReverseName(domain string) string {
	labels := strings.Split(domain, ".")
	for i, j := 0, len(labels)-1; i < j; i, j = i+1, j-1 {
		labels[i], labels[j] = labels[j], labels[i]
	}
	return strings.Join(labels, ".")
}

// PrintMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	bToGB := func(b uint64) uint64 {
		return b / 1024 / 1024
	}

	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	log.Printf("Alloc = %d GB\tTotalAlloc = %d GB\tSys = %d GB\t NumGC = %d",
		bToGB(m.Alloc), bToGB(m.TotalAlloc), bToGB(m.Sys), m.NumGC)
}

func process(ch chan data, targetDate time.Time, done chan bool) {
	earliestIssuance := targetDate.Add(-24 * 90 * time.Hour)
	serialCount := make(map[string]struct{})
	names := make(map[string]struct{})
	for d := range ch {
		date, err := time.Parse(dateFormatFull, d.date)
		if err != nil {
			log.Fatal(err)
		}
		if !(date.After(earliestIssuance) && date.Before(targetDate)) {
			continue
		}
		// de-duplicate the serial numbers and FQDNs
		serialCount[string(d.serialBytes[6:])] = struct{}{}
		names[ReverseName(d.reversedName)] = struct{}{}
	}

	fqdnCount := len(names)
	// Now, having recorded the number of FQDNs, make the fqdnCount map smaller
	// by deleting each FQDN and adding its registered domain. This gets us the
	// count of unique registered domains.
	for k, _ := range names {
		eTLDPlusOne, err := publicsuffix.EffectiveTLDPlusOne(k)
		// EffectiveTLDPlusOne errors when its input is exactly equal to a public
		// suffix, which sometimes happens. In that case, just count the name
		// itself.
		if err != nil {
			continue
		}
		delete(names, k)
		names[eTLDPlusOne] = struct{}{}
	}
	PrintMemUsage()
	registeredDomainCount := len(names)

	targetDateFormatted := targetDate.Format(dateFormat)
	// certsIssued, certsActive, fqdnsActive, regDomainsActive
	fmt.Printf("%s\tNULL\t%d\t%d\t%d\n", targetDateFormatted,
		len(serialCount), fqdnCount, registeredDomainCount)
	done <- true
}

func main() {
	startDateFlag := flag.String("startDate", "", "Start date")
	endDateFlag := flag.String("endDate", "", "End date")
	flag.Parse()
	if startDateFlag == nil || endDateFlag == nil {
		log.Fatal("requires target date")
	}

	startDate, err := time.Parse(dateFormat, *startDateFlag)
	if err != nil {
		log.Fatal(err)
	}

	endDate, err := time.Parse(dateFormat, *endDateFlag)
	if err != nil {
		log.Fatal(err)
	}

	for d := startDate; d.Before(endDate); d = d.Add(24 * time.Hour) {
		doDate(d)
	}
}

func doDate(targetDate time.Time) {
	var files []string

	for d := targetDate.Add(-90 * 24 * time.Hour); d.Before(targetDate.Add(24 * time.Hour)); d = d.Add(24 * time.Hour) {
		files = append(files, d.Format(dateFormat+".tsv"))
	}
	done := make(chan bool)
	ch := make(chan data, 100000)

	go process(ch, targetDate, done)

	go read(ch, files)
	<-done
}

func read(ch chan data, filenames []string) {
	for _, filename := range filenames {
		f, err := os.Open(filename)
		if err != nil {
			log.Print(err)
			continue
		}
		reader := csv.NewReader(f)
		reader.Comma = '\t'

		for {
			record, err := reader.Read()
			if err != nil {
				break
			}
			reversedName, date, serial := record[1], record[2], record[3]
			serialBytes, err := hex.DecodeString(serial)
			if err != nil {
				log.Fatal(err)
			}
			ch <- data{date: date, serialBytes: serialBytes, reversedName: reversedName}
		}
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
	}
	close(ch)
	select {}
}
