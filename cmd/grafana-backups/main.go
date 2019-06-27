package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/letsencrypt/sre-tools/cmd"
)

var timeout, _ = time.ParseDuration("15s")

const (
	dashboardsByUID = "/api/dashboards/uid/"
	allDashboards   = "/api/search?dash-db"
)

// writeDashboardFile queries the Grafana instance by a dashboard's UID for
// the raw JSON and writes the result to a specified directory
func writeDashboardFile(outputDirectory, uid, url, apiKey string) error {
	body, err := fetch(url+dashboardsByUID+uid, apiKey)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(outputDirectory, uid+".json"), body, 0644)
}

// fetch uses a fully qualified URL and api key to query a Grafana instance
// for information at a given path. It returns the resulting body
func fetch(url, apiKey string) ([]byte, error) {
	//	timeout, _ := time.ParseDuration("15s")
	client := http.Client{
		Timeout: timeout,
	}

	request, err := http.NewRequest("GET", url, nil)
	request.Header.Set("Authorization", "Bearer "+apiKey)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Invalid response code %d: %s", resp.StatusCode, body)
	}
	return body, nil
}

// checkEnv checks that the environment variables are set and returns an error
// if they are not. We need to use an API Key to access Grafana and the cleanest
// way to set it is with an environment variable so it won't be visible in the
// process flags. While we're creating one environment variable, we might as
// well set the remaining variables instead of having a mix of environment
// variables and flags.
func checkEnv() error {
	if os.Getenv("GRAFANA_URL") == "" || os.Getenv("GRAFANA_API_KEY") == "" || os.Getenv("GRAFANA_BACKUP_DIR") == "" {
		return errors.New("Environment variables GRAFANA_URL, GRAFANA_API_KEY, and GRAFANA_BACKUP_DIR must be set")
	}
	return nil
}

func main() {
	err := checkEnv()
	cmd.FailOnError(err, "environment variables")
	url := os.Getenv("GRAFANA_URL")
	apiKey := os.Getenv("GRAFANA_API_KEY")
	backupDir := os.Getenv("GRAFANA_BACKUP_DIR")

	body, err := fetch(url+allDashboards, apiKey)
	cmd.FailOnError(err, "fetching dashboards")

	type dbItem struct {
		UID string
	}

	var items []dbItem
	err = json.Unmarshal(body, &items)
	cmd.FailOnError(err, "Unmarshalling JSON body")

	if len(items) == 0 {
		os.Exit(1)
	}

	for _, dashboard := range items {
		err := writeDashboardFile(backupDir, dashboard.UID, url, apiKey)
		cmd.FailOnError(err, "Writing Dashboard fles")
	}
}
