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

func writeDashboardFile(uid string) error {
	body, err := fetch("/api/dashboards/uid/" + uid)
	if err != nil {
		return err
	}

	return ioutil.WriteFile("output/"+uid+".json", body, 0644)
}

func fetch(path string) ([]byte, error) {
	if os.Getenv("GRAFANA_URL") == "" || os.Getenv("GRAFANA_API_KEY") == "" {
		return nil, fmt.Errorf("Environment variables GRAFANA_URL and GRAFANA_API_KEY must be set")
	}
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

func main() {
	body, err := fetch("/api/search?dash-db")
	cmd.FailOnError(err, "fetching dashboards")
	type dbItem struct {
		UID string
	}

	var items []dbItem
	err = json.Unmarshal(body, &items)
	cmd.FailOnError(err, "Unmarshalling JSON body")

	for _, dashboard := range items {
		writeDashboardFile(dashboard.UID)
	}

}
