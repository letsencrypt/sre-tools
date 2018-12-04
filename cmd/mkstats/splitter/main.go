// This binary reads a TSV on stdin that has an ISO 8601 date in the third
// field, then splits up the lines into files based on the day of that date.
package main

import (
	"encoding/csv"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	var err error
	reader := csv.NewReader(os.Stdin)
	reader.Comma = '\t'
	files := make(map[string]*os.File)
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		date := record[2][:10]
		_, err = time.Parse("2006-01-02", date)
		if err != nil {
			log.Fatal(err)
		}
		if _, ok := files[date]; !ok {
			f, err := os.OpenFile(date+".tsv", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
			if err != nil {
				log.Fatal(err)
			}
			files[date] = f
		}
		_, err = files[date].WriteString(strings.Join(record, "\t") + "\n")
		if err != nil {
			log.Fatal(err)
		}
	}
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}
}
