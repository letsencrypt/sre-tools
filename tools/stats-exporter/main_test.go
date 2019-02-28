package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
)

type oneRow struct {
	id, rname, notBefore, serial string
}
type myRows struct {
	rows []oneRow
}

func (m *myRows) Next() bool {
	if len(m.rows) > 0 {
		return true
	}
	return false
}

func (m *myRows) Scan(dest ...interface{}) error {
	if len(dest) != 4 {
		return fmt.Errorf("wrong number of dest: %d", len(dest))
	}
	*(dest[0].(*string)) = m.rows[0].id
	*(dest[1].(*string)) = m.rows[0].rname
	*(dest[2].(*string)) = m.rows[0].notBefore
	*(dest[3].(*string)) = m.rows[0].serial
	m.rows = m.rows[1:]
	return nil
}

// construct a slice of rows and test that they are correctly written in TSV
// format by the writeTSVData function
func TestWriteTSVData(t *testing.T) {
	var testData = &myRows{
		rows: []oneRow{
			oneRow{
				id:        "1",
				rname:     "com.example",
				notBefore: "2019-01-01 01:00:00",
				serial:    "abc",
			},
			oneRow{
				id:        "2",
				rname:     "com.example",
				notBefore: "2019-01-01 01:00:00",
				serial:    "def",
			},
			oneRow{
				id:        "3",
				rname:     "com.example",
				notBefore: "2019-01-01 01:00:00",
				serial:    "ghi",
			},
		},
	}
	var buf bytes.Buffer
	err := writeTSVData(testData, &buf)
	if err != nil {
		t.Fatalf("writing tsv: %s", err)
	}

	expected := `1	com.example	2019-01-01 01:00:00	abc
2	com.example	2019-01-01 01:00:00	def
3	com.example	2019-01-01 01:00:00	ghi
`
	if !bytes.Equal([]byte(expected), buf.Bytes()) {
		t.Errorf("incorrect output: expected %q, got %q", expected, buf.Bytes())
	}

}

type errorRows struct {
}

func (e *errorRows) Next() bool {
	return true
}

func (e *errorRows) Scan(dest ...interface{}) error {
	return fmt.Errorf("I always error")
}

func TestWriteTSVDataError(t *testing.T) {
	var buf bytes.Buffer
	err := writeTSVData(&errorRows{}, &buf)
	if err == nil {
		t.Errorf("expected error")
	}
}

type errorWriter struct {
}

func (e *errorWriter) Write(p []byte) (int, error) {
	return 0, fmt.Errorf("this is actually an error")
}

// test writing the TSV file and experiencing an error while writing
// to the file
func TestWriterError(t *testing.T) {
	var testData = &myRows{
		rows: []oneRow{
			oneRow{
				id:        "1",
				rname:     "com.example",
				notBefore: "2019-01-01 01:00:00",
				serial:    "abc",
			},
		},
	}
	err := writeTSVData(testData, &errorWriter{})
	if err == nil {
		t.Errorf("expected error")
	}
	if !strings.Contains(err.Error(), "this is actually an error") {
		t.Errorf("wrong error. got: %q", err)
	}
}

type simpleDB struct {
}

func (s *simpleDB) Query(string, ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func TestQueryDB(t *testing.T) {
	content := []byte("some@tcp(fake:3306)/DSN data")
	tmpfile, err := ioutil.TempFile("", "")

	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}
	// since we cannot construct a fake database object to query against
	// we check the connection params and return a simpleDB that can
	// be used with the Query function to run the query against
	checkedSQLOpen := func(driver, dsn string) (dbQueryable, error) {
		if driver != "mysql" {
			return nil, fmt.Errorf("wrong driver %s", driver)
		}
		if dsn != string(content) {
			return nil, fmt.Errorf("wrong dsn %s", dsn)
		}
		return &simpleDB{}, nil
	}
	savedSQLOpen := sqlOpen
	sqlOpen = checkedSQLOpen
	defer func() {
		sqlOpen = savedSQLOpen
	}()

	_, err = queryDB(tmpfile.Name(), "2019-01-01", "2019-01-02")
	if err != nil {
		t.Fatal(err)
	}

}

type errorDB struct {
}

func (s *errorDB) Query(string, ...interface{}) (*sql.Rows, error) {
	return nil, fmt.Errorf("this is actually an error")
}

func TestQueryDBError(t *testing.T) {
	content := []byte("some@tcp(fake:3306)/DSN data")
	tmpfile, err := ioutil.TempFile("", "")

	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}
	checkedSQLOpen := func(driver, dsn string) (dbQueryable, error) {
		return &errorDB{}, nil
	}
	savedSQLOpen := sqlOpen
	sqlOpen = checkedSQLOpen
	defer func() {
		sqlOpen = savedSQLOpen
	}()
	_, err = queryDB(tmpfile.Name(), "2019-01-01", "2019-01-02")
	if err == nil {
		t.Errorf("expected error")
	}
	if !strings.Contains(err.Error(), "this is actually an error") {
		t.Errorf("wrong error. got: %q", err)
	}
}

type errorReadFile struct {
}

func TestQueryDBConnectError(t *testing.T) {
	_, err := queryDB("nonExistentFile", "2019-01-01", "2019-01-02")
	if err == nil {
		t.Errorf("expected error")
	}
	if !strings.Contains(err.Error(), "Could not open database connection file") {
		t.Errorf("wrong error. got: %q", err)
	}
}

// test the exec commands by checking the command string that will be
// executed in a shell
func TestCompress(t *testing.T) {
	outputFileName := "fakeFile.tsv"

	checkedArgs := func(c *exec.Cmd) error {
		expected := "gzip fakeFile.tsv"
		args := strings.Join(c.Args, " ")
		if args != expected {
			return fmt.Errorf("wrong argument string. Got %q expected %q", args, expected)
		}
		return nil
	}
	savedExecRun := execRun
	execRun = checkedArgs
	defer func() {
		execRun = savedExecRun
	}()
	err := compress(outputFileName)
	if err != nil {
		t.Fatal(err)
	}

}

func TestScp(t *testing.T) {
	outputFileName := "fakeFile.tsv"
	destination := "localhost:/tmp"
	key := "id_rsa"

	checkedArgs := func(c *exec.Cmd) error {
		expected := "scp -i id_rsa fakeFile.tsv.gz localhost:/tmp"
		args := strings.Join(c.Args, " ")
		if args != expected {
			return fmt.Errorf("wrong argument string. Got %q expected %q", args, expected)
		}
		return nil
	}
	savedExecRun := execRun
	execRun = checkedArgs
	defer func() {
		execRun = savedExecRun
	}()
	err := scp(outputFileName, destination, key)
	if err != nil {
		t.Fatal(err)
	}
}
