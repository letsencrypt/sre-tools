package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jmhodges/clock"
	"github.com/letsencrypt/boulder/db"
	"github.com/letsencrypt/boulder/policy"
)

type emailAuditor struct {
	dbMap        *db.WrappedMap
	clk          clock.Clock
	grace        time.Duration
	queryResults []queryResult
}

type queryResult struct {
	ID      int
	Contact []byte
}

func (r *queryResult) getAddresses() ([]string, error) {
	var contactFields []string
	var addresses []string
	err := json.Unmarshal(r.Contact, &contactFields)
	if err != nil {
		return nil, err
	}
	for _, entry := range contactFields {
		if strings.HasPrefix(entry, "mailto:") {
			addresses = append(addresses, strings.TrimPrefix(entry, "mailto:"))
		}
	}
	return addresses, nil
}

func (m *emailAuditor) collectEmails() error {
	_, err := m.dbMap.Select(
		m.queryResults,
		`SET SESSION TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;
		SELECT DISTINCT
			r.id AS id,
		    r.contact AS contact
	    FROM registrations AS r
		    INNER JOIN certificates AS c on c.registrationID = r.id
	    WHERE r.contact != '[]'
		    AND c.expires >= :expireCutoff`,
		map[string]interface{}{
			"expireCutoff": m.clk.Now().Add(-m.grace),
		})
	if err != nil {
		return fmt.Errorf("error while querying: %s", err)
	}
	return nil
}

func (e *emailAuditor) run() error {
	err := e.collectEmails()
	if err != nil {
		return err
	}
	for _, result := range e.queryResults {
		addresses, err := result.getAddresses()
		if err != nil {
			return err
		}
		for _, address := range addresses {
			err := policy.ValidEmail(address)
			if err != nil {
				fmt.Printf("%s", address)
				continue
			}
		}
	}
	return nil
}

func main() {
	auditor := emailAuditor{grace: 2 * 24 * time.Hour}
	err := auditor.run()
	if err != nil {
		log.Fatalf("Problem encountered while running audit: %s", err)
	}
}
