package main

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestWriteDashboardFile(t *testing.T) {
	expected := "Successful Response"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/dashboards/uid/fakeUID" {

			_, err := io.WriteString(w, expected)

			if err != nil {
				t.Fatal(err)
			}
		} else {
			_, err := io.WriteString(w, "Bad response for providing an incorrect path")

			if err != nil {
				t.Fatal(err)
			}
		}
	}))
	defer ts.Close()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	err = writeDashboardFile(dir, "fakeUID", ts.URL, "test")
	if err != nil {
		t.Errorf("writeDashboardFile failed")
	}
	content, err := ioutil.ReadFile(dir + "/fakeUID.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "Successful Response" {
		t.Errorf("Expected %q. Got %q", expected, string(content))
	}
}

func TestWriteDashboardFileError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer ts.Close()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	err = writeDashboardFile(dir, "FakeUID", ts.URL, "test")
	if err == nil {
		t.Error("Expected error, got none")
	}
	if !strings.Contains(err.Error(), "Invalid response code") {
		t.Errorf("Expected 'Invalid response code', got %q", err)
	}
	_, err = os.Stat(dir + "/FakeUID")
	if err == nil {
		t.Error("Expected error, got none")
	}
	if !os.IsNotExist(err) {
		t.Errorf("Expected `no such file or directory`  got %q", err)
	}
}

func TestCheckEnvironment(t *testing.T) {
	os.Setenv("GRAFANA_URL", "fakeserver.com")
	os.Setenv("GRAFANA_API_KEY", "test")
	os.Setenv("GRAFANA_BACKUP_DIR", "/tmp/backup")

	if err := checkEnv(); err != nil {
		t.Errorf("Expected no error, go %q", err)
	}
}

func TestNoEnvironmentSet(t *testing.T) {
	os.Clearenv()
	err := checkEnv()

	if !strings.Contains(err.Error(), "Environment variables") {
		t.Errorf("Expected `Environment variables` got %q", err)
	}
}

func TestFetchGoodAuthorization(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header["Authorization"][0]
		if !strings.HasPrefix(authorization, "Bearer fake key") {
			w.WriteHeader(400)
		}
	}))
	defer ts.Close()

	if _, err := fetch("/grafana", ts.URL, "fake key"); err != nil {
		t.Fatal(err)
	}
}

func TestFetchBadAuthorization(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header["Authorization"][0]
		if !strings.HasPrefix(authorization, "Bearer fake key") {
			w.WriteHeader(400)
		}
	}))
	defer ts.Close()
	var err error

	if _, err = fetch("/grafana", ts.URL, "bad key"); err == nil {
		t.Fatal("expected error, got none")
	}

	if !strings.Contains(err.Error(), "Invalid response code") {
		t.Errorf("Expected 'invalid response code' got %q", err)
	}
}
