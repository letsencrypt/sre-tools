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
	fmt.Printf("Alloc = %v GB", bToGB(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v GB", bToGB(m.TotalAlloc))
	fmt.Printf("\tSys = %v GB", bToGB(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func process(ch chan data, targetDate time.Time) {
	earliestIssuance := targetDate.Add(-24 * 90 * time.Hour)
	serialCount := make(map[string]struct{})
	reversedNameCount := make(map[string]struct{})
	registeredDomainCount := make(map[string]struct{})
	for d := range ch {
		date, err := time.Parse(dateFormatFull, d.date)
		if err != nil {
			log.Fatal(err)
		}
		if !(date.After(earliestIssuance) && date.Before(targetDate)) {
			continue
		}
		serialCount[string(d.serialBytes[6:])] = struct{}{}
		reversedNameCount[d.reversedName] = struct{}{}
		forwardName := ReverseName(d.reversedName)
		eTLDPlusOne, err := publicsuffix.EffectiveTLDPlusOne(forwardName)
		// EffectiveTLDPlusOne errors when its input is exactly equal to a public
		// suffix, which sometimes happens. In that case, just count the name
		// itself.
		if err != nil {
			eTLDPlusOne = forwardName
		}
		registeredDomainCount[eTLDPlusOne] = struct{}{}
	}

	targetDateFormatted := targetDate.Format(dateFormat)
	fmt.Printf("serials %s\t%d\n", targetDateFormatted, len(serialCount))
	fmt.Printf("names %s\t%d\n", targetDateFormatted, len(reversedNameCount))
	fmt.Printf("registered %s\t%d\n", targetDateFormatted, len(registeredDomainCount))
	PrintMemUsage()
	os.Exit(0)
}

func main() {
	targetDateFlag := flag.String("targetDate", "", "Target date")
	flag.Parse()
	if targetDateFlag == nil {
		log.Fatal("requires target date")
	}

	targetDate, err := time.Parse(dateFormat, *targetDateFlag)
	if err != nil {
		log.Fatal(err)
	}

	ch := make(chan data, 100000)

	go process(ch, targetDate)

	read(ch, flag.Args()[1:])
}

func read(ch chan data, filenames []string) {
	for _, filename := range flag.Args()[1:] {
		f, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
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
