package main

import (
	"encoding/csv"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

type data struct {
	date        string
	serialBytes []byte
}

const dateFormat = "2006-01-02"
const dateFormatFull = "2006-01-02 15:04:05"

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
	earliestIssuance := targetDate.Add(-24 * 90 * time.Hour)

	ch := make(chan data, 100000)
	serialCount := make(map[string]struct{})

	go func() {
		for d := range ch {
			date, err := time.Parse(dateFormatFull, d.date)
			if err != nil {
				log.Fatal(err)
			}
			if !(date.After(earliestIssuance) && date.Before(targetDate)) {
				continue
			}
			serialCount[string(d.serialBytes[6:])] = struct{}{}
		}

		fmt.Printf("%s\t%d\n", *targetDateFlag, len(serialCount))
		os.Exit(0)
	}()

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
			_, date, serial := record[1], record[2], record[3]
			serialBytes, err := hex.DecodeString(serial)
			if err != nil {
				log.Fatal(err)
			}
			ch <- data{date: date, serialBytes: serialBytes}
		}
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
	}
	close(ch)
	select {}
}
