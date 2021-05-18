package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/jmhodges/clock"
	"github.com/letsencrypt/boulder/cmd"
	"github.com/letsencrypt/boulder/db"
	"github.com/letsencrypt/boulder/policy"
	"github.com/letsencrypt/boulder/sa"
)

type contactAuditor struct {
	dbMap *db.WrappedMap
	clk   clock.Clock
	grace time.Duration
}

type queryResult struct {
	ID        int64
	Contact   []byte
	addresses []string
}

type queryResults []*queryResult

func (m contactAuditor) collectEmails() (queryResults, error) {
	var holder queryResults
	_, err := m.dbMap.Exec("SET SESSION TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;")
	if err != nil {
		return nil, fmt.Errorf("error while setting tranaction level: %s", err)
	}
	_, err = m.dbMap.Select(
		&holder,
		`SELECT DISTINCT r.id, r.contact
	    FROM registrations AS r
		    INNER JOIN certificates AS c on c.registrationID = r.id
	    WHERE r.contact != '[]'
		    AND c.expires >= :expireCutoff`,
		map[string]interface{}{
			"expireCutoff": m.clk.Now().Add(-m.grace),
		})
	if err != nil {
		return nil, fmt.Errorf("error while querying database: %s", err)
	}
	return holder, nil
}

func (r *queryResult) unmarshalAddresses() error {
	var contactFields []string
	err := json.Unmarshal(r.Contact, &contactFields)
	if err != nil {
		return err
	}
	for _, entry := range contactFields {
		if strings.HasPrefix(entry, "mailto:") {
			email := strings.TrimPrefix(entry, "mailto:")
			r.addresses = append(r.addresses, email)
		}
	}
	return nil
}

func (e contactAuditor) run() (queryResults, error) {
	results, err := e.collectEmails()
	if err != nil {
		return nil, err
	}
	for _, result := range results {
		err = result.unmarshalAddresses()
		if err != nil {
			return nil, err
		}
		for _, address := range result.addresses {
			err := policy.ValidEmail(address)
			if err != nil {
				fmt.Printf("address: %q for ID: %d failed validation due to: %s\n", address, result.ID, err)
				continue
			}
		}
	}
	return results, nil
}

func main() {
	type config struct {
		ContactAuditor struct {
			DB cmd.DBConfig
			cmd.PasswordConfig
			Features map[string]bool
		}
	}
	configFile := flag.String("config", "", "File containing a JSON config.")
	flag.Parse()

	configData, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("Error while reading %q: %s\n", *configFile, err)
	}

	var cfg config
	err = json.Unmarshal(configData, &cfg)
	if err != nil {
		log.Fatalf("Error while unmarshaling config: %s\n", err)
	}

	dbURL, err := cfg.ContactAuditor.DB.URL()
	if err != nil {
		log.Fatalln("Couldn't load DB URL")
	}

	dbSettings := sa.DbSettings{
		MaxOpenConns: 10,
	}

	dbMap, err := sa.NewDbMap(dbURL, dbSettings)
	if err != nil {
		log.Fatalln("Could not connect to database")
	}

	auditor := contactAuditor{grace: 2 * 24 * time.Hour, clk: clock.New(), dbMap: dbMap}
	_, err = auditor.run()
	if err != nil {
		log.Fatalf("Problem encountered while running audit: %s\n", err)
	}
}
