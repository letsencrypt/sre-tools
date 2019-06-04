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
		_, err := io.WriteString(w, expected)
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	os.Setenv("GRAFANA_URL", ts.URL)
	os.Setenv("GRAFANA_API_KEY", "test")

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	err = writeDashboardFile(dir, "fakeUID")
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

	os.Setenv("GRAFANA_URL", ts.URL)
	os.Setenv("GRAFANA_API_KEY", "test")

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	err = writeDashboardFile(dir, "FakeUID")
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
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Errorf("Expected `no such file or directory`  got %q", err)
	}
}

func TestCheckEnvironment(t *testing.T) {
	os.Setenv("GRAFANA_URL", "fakeserver.com")
	os.Setenv("GRAFANA_API_KEY", "test")
	os.Setenv("GRAFANA_BACKUP_DIR", "/tmp/backup")

	err := checkEnv()
	if err != nil {
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
