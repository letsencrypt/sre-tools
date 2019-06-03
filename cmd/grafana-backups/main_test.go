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
		io.WriteString(w, expected)
	}))
	defer ts.Close()

	os.Setenv("GRAFANA_URL", ts.URL)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	err = writeDashboardFile(dir, "jilliansFakeUID")
	if err != nil {
		t.Errorf("writeDashboardFile failed")
	}
	content, err := ioutil.ReadFile(dir + "/jilliansFakeUID.json")
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

	os.Setenv("GRAFANA_URL", ts.URL)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	err = writeDashboardFile(dir, "jilliansFakeUID")
	if err == nil {
		t.Error("Expected error, got none")
	}
	if !strings.Contains(err.Error(), "Invalid response code") {
		t.Errorf("Expected 'Invalid response code', got %q", err)
	}
	_, err = os.Stat(dir + "/jilliansFakeUID")
	if err == nil {
		t.Error("Expected error, got none")
	}
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Errorf("Expected `no such file or directory`  got %q", err)
	}
}

func TestFetchNoEnvironmentSet(t *testing.T) {
	os.Clearenv()
	_, err := fetch("grafana")

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

	os.Setenv("GRAFANA_URL", ts.URL)
	os.Setenv("GRAFANA_API_KEY", "fake key")

	_, err := fetch("/grafana")
	if err != nil {
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

	os.Setenv("GRAFANA_URL", ts.URL)
	os.Setenv("GRAFANA_API_KEY", "bad key")

	_, err := fetch("/grafana")
	if err == nil {
		t.Fatal("expected error, got none")
	}
	if !strings.Contains(err.Error(), "Invalid response code") {
		t.Errorf("Expected 'invalid response code' got %q", err)
	}
}
