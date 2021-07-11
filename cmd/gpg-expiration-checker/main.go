package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

var (
	pubKey = flag.String("pub_key", "", "Location of GPG key")
)

func recordMetrics() {
	go func() {
		// Read in the GPG key
		gpgEntity, err := readEntity("le.gpg")
		if err != nil {
			fmt.Println(err)
			return
		}

		// Process the primary key
		pk := gpgEntity.PrimaryKey
		pkKeyId := strings.ToUpper(strconv.FormatUint(pk.KeyId, 16))

		// golang crypto/opengpg doesn't export the primary key lifetime without digging into bytes
		// The documentation states: "contains filtered or unexported fields"
		// https://godoc.org/golang.org/x/crypto/openpgp/packet#PublicKey
		pkExpiryDate := pk.CreationTime.AddDate(7, 0, 299).Add(time.Hour * 22)
		prom_processedKeys.WithLabelValues(strconv.FormatBool(pk.IsSubkey), pkKeyId, pk.CreationTime.String(), pkExpiryDate.String()).Inc()

		// Process all subkeys
		for _, sk := range gpgEntity.Subkeys {
			skExpiryDate := sk.PublicKey.CreationTime.Add(time.Duration(*sk.Sig.KeyLifetimeSecs) * time.Second)
			skKeyId := strings.ToUpper(strconv.FormatUint(*sk.Sig.IssuerKeyId, 16))
			prom_processedKeys.WithLabelValues(strconv.FormatBool(sk.PublicKey.IsSubkey), skKeyId, sk.Sig.CreationTime.String(), skExpiryDate.String()).Inc()
		}
	}()
}

var (
	prom_processedKeys = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gpg_key_expiration_seconds",
		Help: "Outputs number of seconds until expiration of the primary key and any subkeys.",
	}, []string{"isSubKey", "keyid", "creation", "expiration"})
)

func DownloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	output, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = io.Copy(output, resp.Body)
	return err
}

func readEntity(name string) (*openpgp.Entity, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	block, err := armor.Decode(f)
	if err != nil {
		return nil, err
	}
	return openpgp.ReadEntity(packet.NewReader(block.Body))
}

func main() {
	// TODO: Check if no flags exist
	flag.Parse()

	// TODO: Store the gpg key in memory
	if err := DownloadFile("le.gpg", *pubKey); err != nil {
		panic(err)
	}

	recordMetrics()

	// TODO: Pick a better port rather than the default example
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
