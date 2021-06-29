package main

import (
	"fmt"
	"log"

	"github.com/letsencrypt/sre-tools/cmd/compliance-tracker/reviews"
)

func main() {
	mdspReviews, err := reviews.GetMDSPReviewsForWeek()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("\n### New MDSP Threads\n")
	for _, entry := range mdspReviews {
		fmt.Print(entry.String())
	}

	mzReviews, err := reviews.GetMozDevCAReviewsForWeek()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("\n### New Mozilla CA Compliance Bugs\n")
	for _, entry := range mzReviews {
		fmt.Print(entry.String())
	}

	leReviews, err := reviews.GetMozLEReviewsForWeek()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("\n### Open Let's Encrypt Bugs\n")
	for _, entry := range leReviews {
		fmt.Print(entry.String())
	}
}
