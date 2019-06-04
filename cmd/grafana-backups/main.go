package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/letsencrypt/sre-tools/cmd"
)

// Query the Grafana instance by a dashboard's UID for the raw JSON and write
// the result to a specified directory
func writeDashboardFile(outputDirectory, uid string) error {
	body, err := fetch("/api/dashboards/uid/" + uid)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(outputDirectory+"/"+uid+".json", body, 0644)
}

// Use environment variables to set the Grafana instance URL and the API key
// to prevent providing the key on the command line. We then query the instance
// for a given path and return the resulting body.
func fetch(path string) ([]byte, error) {
	timeout := time.Duration(15 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}

	request, err := http.NewRequest("GET", os.Getenv("GRAFANA_URL")+path, nil)
	request.Header.Set("Authorization", "Bearer "+os.Getenv("GRAFANA_API_KEY"))
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

// We need to use an API Key to access Grafana and the cleanest way to set it
// is with an environment variable so it won't be visible in the process flags.
// While we're creating one environment variable, we might as well set the
// remaining variables instead of having a mix of environment variables and flags.
func checkEnv() error {
	if os.Getenv("GRAFANA_URL") == "" || os.Getenv("GRAFANA_API_KEY") == "" || os.Getenv("GRAFANA_BACKUP_DIR") == "" {
		return fmt.Errorf("Environment variables GRAFANA_URL and GRAFANA_API_KEY and GRAFANA_BACKUP_DIR must be set")
	}
	return nil
}

func main() {
	err := checkEnv()
	cmd.FailOnError(err, "environment variables")

	body, err := fetch("/api/search?dash-db")
	cmd.FailOnError(err, "fetching dashboards")
	type dbItem struct {
		UID string
	}

	var items []dbItem
	err = json.Unmarshal(body, &items)
	cmd.FailOnError(err, "Unmarshalling JSON body")

	for _, dashboard := range items {
		writeDashboardFile("backup", dashboard.UID)
	}

}
