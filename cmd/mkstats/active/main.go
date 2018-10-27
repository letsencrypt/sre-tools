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
	targetEnd := targetDate.Add(24 * 90 * time.Hour)

	var err error
	reader := csv.NewReader(os.Stdin)
	reader.Comma = '\t'
	ch := make(chan data, 100000)
	serialCount := make(map[string]map[string]struct{})
	go func() {
		for d := range ch {
			date, err := time.Parse(dateFormat, *d.date)
			if err != nil {
				log.Fatal(err)
			}
			if date.Before(targetDate) || date.After(targetEnd) {
				continue
			}
			if serialCount[d.date] == nil {
				serialCount[d.date] = make(map[string]struct{})
			}
			serialCount[d.date][string(d.serialBytes[6:])] = struct{}{}
		}

		for k, v := range serialCount {
			fmt.Printf("%s\t%d\n", k, len(v))
		}
		os.Exit(0)
	}()

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		_, date, serial := record[1], record[2], record[3]
		date = date[:10]
		serialBytes, err := hex.DecodeString(serial)
		if err != nil {
			log.Fatal(err)
		}
		ch <- data{date: date, serialBytes: serialBytes}
	}
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}
	close(ch)
	select {}
}
