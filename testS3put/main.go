package main

import (
	"log"

	"github.com/letsencrypt/sre-tools/s3Put"
)

// Push compressed file to an aws s3 bucket. Requires filename, aws region
// and s3 bucket name.
func pushToS3(s3Region, s3Bucket, outputFileName string) error {
	err := s3Put.AddFileToS3(s3Region, s3Bucket, outputFileName)
	return err
}

func main() {
	err := pushToS3("us-east-1", "ag-test-put", "other/results-2019-08-04.tsv.gz")
	if err != nil {
		log.Fatalf("Error: %q", err)
	}
}
