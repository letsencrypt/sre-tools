package main

import (
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
)

type data struct {
	date        string
	serialBytes []byte
}

func main() {
	var err error
	reader := csv.NewReader(os.Stdin)
	reader.Comma = '\t'
	ch := make(chan data, 100000)
	go func() {
		serialCount := make(map[string]map[string]struct{})

		for d := range ch {
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
