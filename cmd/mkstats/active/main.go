package main

import (
	"encoding/binary"
	"encoding/csv"
	"encoding/hex"
	"flag"
	"fmt"
	"hash/fnv"
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

const expectedSize = 5e7

func process(ch chan data, targetDate time.Time, done chan bool) {
	earliestIssuance := targetDate.Add(-24 * 90 * time.Hour)
	serialCount := make(map[uint64]struct{}, expectedSize)
	names := make(map[uint64]struct{}, expectedSize)
	registeredNames := make(map[uint64]struct{}, expectedSize)
	today := make(map[uint64]struct{}, 1e6)
	for d := range ch {
		date, err := time.Parse(dateFormatFull, d.date)
		if err != nil {
			log.Fatal(err)
		}
		if !(date.After(earliestIssuance) && date.Before(targetDate)) {
			continue
		}
		// de-duplicate the serial numbers and FQDNs
		serialUint64 := binary.BigEndian.Uint64(d.serialBytes[6:])
		serialCount[serialUint64] = struct{}{}

		if date.After(targetDate.Add(-24*time.Hour)) && date.Before(targetDate) {
			today[serialUint64] = struct{}{}
		}

		hasher1 := fnv.New64a()
		hasher1.Write([]byte(d.reversedName))
		nameUint64 := hasher1.Sum64()
		names[nameUint64] = struct{}{}
		eTLDPlusOne, err := publicsuffix.EffectiveTLDPlusOne(ReverseName(d.reversedName))
		// EffectiveTLDPlusOne errors when its input is exactly equal to a public
		// suffix, which sometimes happens. In that case, just count the name
		// itself.
		if err != nil {
			eTLDPlusOne = ReverseName(d.reversedName)
		}
		hasher2 := fnv.New64a()
		hasher2.Write([]byte(eTLDPlusOne))
		eTLDPlusOneUint64 := hasher2.Sum64()
		registeredNames[eTLDPlusOneUint64] = struct{}{}
	}

	PrintMemUsage()

	targetDateFormatted := targetDate.Format(dateFormat)
	// certsIssued, certsActive, fqdnsActive, regDomainsActive
	fmt.Printf("%s\t%d\t%d\t%d\t%d\n", targetDateFormatted, len(today),
		len(serialCount), len(names), len(registeredNames))
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
