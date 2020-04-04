package main

import (
	"crypto/x509"
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/letsencrypt/boulder/goodkey"
)

// We only use these two functions on the sql.rows object, so we just define an
// interface with those methods instead of importing all of them. This facilitates
// mock implementation for unit tests
type sqlRows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close() error
}

// dbQueryable is an interface for the sql.Query function that is needed to
// query the database. Using this interface allows tests to swap out the
// query implementation and return the needed object type since we cannot
// create a sql.Rows sturct to test on
type dbQueryable interface {
	Query(string, ...interface{}) (*sql.Rows, error)
	Close() error
}

// Used to enable unit tests on the sql.Open function and return the interface
// needed to execute the Query commands. In unit tests, we can mock this
// function and return the dbQueryable type and eliminate the need for having
// a live database up when tests run or mocking the rows
var sqlOpen = func(driver, dsn string) (dbQueryable, error) {
	return sql.Open(driver, dsn)
}

var batchSize = flag.Int("batchSize", 1000, "Size of batch to query the database with.")

func main() {
	dbConnect := flag.String("dbConnect", "", "Path to the DB URL file")
	blockedKeysFile := flag.String("blockedKeysFile", "", "Path to blocked key file")
	startingID := flag.Int("startingID", 0, "ID to start iterating on the certificates table from")

	flag.Parse()
	if *dbConnect == "" || *blockedKeysFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	keyPolicy, err := goodkey.NewKeyPolicy("", *blockedKeysFile)
	if err != nil {
		log.Fatal(err)
	}

	dbDSN, err := ioutil.ReadFile(*dbConnect)
	if err != nil {
		log.Fatalf("Could not open database connection file %q: %s", dbConnect, err)
	}
	db, err := sqlOpen("mysql", strings.TrimSpace(string(dbDSN)))
	defer func() {
		_ = db.Close()
	}()
	if err != nil {
		log.Fatalf("Could not establish database connection: %s", err)
	}

	maxID := *startingID
	for {
		var err error
		maxID, err = queryOnce(db, keyPolicy, maxID)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("processed batch of certificates, maxID: %d\n", maxID)
	}
}

func queryOnce(db dbQueryable, keyPolicy goodkey.KeyPolicy, maxID int) (int, error) {
	rows, err := db.Query(
		`SELECT id, serial, der
		 FROM certificates
		 where id > ?
		 LIMIT ?`, maxID, *batchSize)
	if err != nil {
		return -1, fmt.Errorf("Could not complete database query: %s", err)
	}
	if rows == nil || !rows.Next() {
		return -1, fmt.Errorf("No results match query for certID > %d (possibly reached latest certificate)", maxID)
	}
	defer func() {
		rows.Close()
	}()

	var (
		id     int
		serial string
		der    []byte
	)
	for {
		if err := rows.Scan(&id, &serial, &der); err != nil {
			return -1, err
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return -1, err
		}
		// If the key is forbidden by the key policy (typically because it's
		// blocked), print the serial and error message to stderr.
		if err := keyPolicy.GoodKey(cert.PublicKey); err != nil {
			fmt.Fprintln(os.Stderr, serial, err)
		}
		if !rows.Next() {
			break
		}
	}
	return id, nil
}
