package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/letsencrypt/sre-tools/cmd"
)

// We only use these two functions on the sql.rows object, so we just define an
// interface with those methods instead of importing all of them. This facilitates
// mock implementation for unit tests
type sqlRows interface {
	Next() bool
	Scan(dest ...interface{}) error
}

// dbQueryable is an interface for the sql.Query function that is needed to
// query the database. Using this interface allows tests to swap out the
// query implementation and return the needed object type since we cannot
// create a sql.Rows sturct to test on
type dbQueryable interface {
	Query(string, ...interface{}) (*sql.Rows, error)
}

// Used to enable unit tests on the sql.Open function and return the interface
// needed to execute the Query commands. In unit tests, we can moock this
// function and return the dbQueryable type and eliminate the need for having
// a live database up when tests run or mocking the rows
var sqlOpen = func(driver, dsn string) (dbQueryable, error) {
	return sql.Open(driver, dsn)
}

// Used to to enable unit tests where we don't want to actually run commands
// on the host. Instead, we can mock the cmd.Run functions and focus on the
// error logic
var execRun = func(c *exec.Cmd) error {
	return c.Run()
}

// Connect to the database and run the select query to gather all of the
// issuedNames between two timestamps. In main() we construct the timeframe as
// 24 hour window covering the previous day. It is expected that this program
// will run after 00:00 on any given day in order to get a complete data set of
// the previous day's issued names.
func queryDB(dbConnect, beginTimeStamp, endTimeStamp string) (*sql.Rows, error) {
	dbDSN, err := ioutil.ReadFile(dbConnect)
	if err != nil {
		return nil, fmt.Errorf("Could not open database connection file %q: %s", dbConnect, err)
	}
	db, err := sqlOpen("mysql", strings.TrimSpace(string(dbDSN)))
	if err != nil {
		return nil, fmt.Errorf("Could not establish database connection: %s", err)
	}
	rows, err := db.Query(
		`SELECT id, reversedName, notBefore, serial
		 FROM issuedNames 
		 where notBefore >= ? and notBefore < ?`, beginTimeStamp, endTimeStamp)
	if err != nil {
		return nil, fmt.Errorf("Could not complete database query: %s", err)
	}
	return rows, nil
}

// Write the query results in TSV format
func writeTSVData(rows sqlRows, outFile io.Writer) error {
	for rows.Next() {
		var (
			id, rname, notBefore, serial string
		)
		if err := rows.Scan(&id, &rname, &notBefore, &serial); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(outFile, "%s\t%s\t%s\t%s\n", id, rname, notBefore, serial); err != nil {
			return err
		}
	}
	return nil
}

// Compress the results TSV file
func compress(outputFileName string) error {
	gzipCmd := exec.Command("gzip", outputFileName)
	if err := execRun(gzipCmd); err != nil {
		return fmt.Errorf("Could not gzip result file: %s", err)
	}
	return nil
}

// SCP the compressed file to a remote host using a specified key file.
// Requiring a key allows low privledge users wihtout a home directory or
// persistent SSH configs to to run the program and transfer the files to
// hosts that have SSH confifugred for a set of authorized keys
func scp(outputFileName, destination, key string) error {
	outputGZIPName := outputFileName + ".gz"
	scpCmd := exec.Command("scp", "-i", key, outputGZIPName, destination)
	if err := execRun(scpCmd); err != nil {
		return fmt.Errorf("Could not scp result file %q to %q: %s", outputFileName, destination, err)
	}
	return nil
}

func main() {
	dbConnect := flag.String("dbConnect", "", "Path to the DB URL file")
	destination := flag.String("destination", "localhost:/tmp", "Location to SCP the gzipped TSV result file to")
	key := flag.String("key", "id_rsa", "Identity key for SCP")
	flag.Parse()

	// The query we run against the database is to examine the previous day of data
	// we construct dates that correspond to the start and stop of that 24 hour window
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	yesterdayDateStamp := yesterday.Format("2006-01-02")
	endDateStamp := now.Format("2006-01-02")
	outputFileName := fmt.Sprintf("results-%s.tsv", yesterdayDateStamp)

	outFile, err := os.OpenFile(outputFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	cmd.FailOnError(err, fmt.Sprintf("Could not create results file %q", outputFileName))

	defer func() {
		err := outFile.Close()
		cmd.FailOnError(err, fmt.Sprintf("Could not close output file %q", outputFileName))
	}()

	rows, err := queryDB(*dbConnect, yesterdayDateStamp, endDateStamp)
	cmd.FailOnError(err, "Could not complete database work")

	err = writeTSVData(rows, outFile)
	cmd.FailOnError(err, "Could not write TSV data")

	err = compress(outputFileName)
	cmd.FailOnError(err, "Could not compress results")
	err = scp(outputFileName, *destination, *key)
	cmd.FailOnError(err, "Could not send results")
}
