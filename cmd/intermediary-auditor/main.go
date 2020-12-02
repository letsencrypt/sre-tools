package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	r3 = "R3"
	x3 = "Let's Encrypt Authority X3"
)

var debugMode bool
var statsCsv string
var wg sync.WaitGroup

// chainContainsR3 checks if a chain of certs contains a certificate
// where the Subject Common Name matches the const of r3
func chainContainsR3(chain []*x509.Certificate) bool {
	for _, cert := range chain[1:] {
		if cert.Subject.CommonName == r3 {
			return true
		}
	}
	return false
}

// chainContainsX3 checks if a chain of certs contains a certificate
// where the Subject Common Name matches the const of x3
func chainContainsX3(chain []*x509.Certificate) bool {
	for _, cert := range chain[1:] {
		if cert.Subject.CommonName == x3 {
			return true
		}
	}
	return false
}

// rawToChain marshals a slice of byte slices representing an x.509
// certificate chain to a slice of *x.509Certificate objects
func rawToChain(rawCerts [][]byte) []*x509.Certificate {
	chain := []*x509.Certificate{}
	for _, rawCert := range rawCerts {
		cert, err := x509.ParseCertificate(rawCert)
		if err != nil {
			continue
		}
		chain = append(chain, cert)
	}
	return chain
}

// chaing2String is used solely if debug is true. Iterates from the
// leaf (end-entity) certificate all the way up the chain building a
// string to represent the Subject Common Name and Issuer Common Name
// for each Certificate
func chaing2String(chain []*x509.Certificate) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("leafCert: [subjectCN: %s | issuerCN: %s]", chain[0].Subject.CommonName, chain[0].Issuer.CommonName))
	for num, cert := range chain[1:] {
		sb.WriteString(fmt.Sprintf(" -> chainCert%d: [subjectCN: %s | issuerCN: %s] ", num, cert.Subject.CommonName, cert.Issuer.CommonName))
	}
	return sb.String()
}

// auditChain for a given slice of byte slices representing an x.509
// certificate chain, if the Issuer Common Name is const x3 or r3,
// validates that the resulting chain of x509 Certificates contains the
// corresponding x3 or r3 intermediate that issued the leaf Certificate.
// If a mis-match is present, a string containing the Subject Common
// Name of the leaf certificate is returned, else, in all other cases an
// empty string is returned.
func auditChain(rawCerts [][]byte) string {
	chain := rawToChain(rawCerts)
	leafIssuerCN := chain[0].Issuer.CommonName
	if len(chain) > 1 {
		if debugMode == true {
			fmt.Println(chaing2String(chain))
		}
		if leafIssuerCN != r3 && leafIssuerCN != x3 {
			return ""
		}
		if leafIssuerCN == r3 && chainContainsR3(chain) {
			return ""
		}
		if leafIssuerCN == x3 && chainContainsX3(chain) {
			return ""
		}
		return chain[0].Subject.CommonName
	}
	return ""
}

// auditHostname for a given hostname, dials and starts a TLS handshake.
// The tls.Config skips verification steps and delegates verfication to
// an anonymous function that audits the certification chain
func auditHostname(hostname string) {
	defer wg.Done()
	dialer := net.Dialer{Timeout: 1 * time.Second}
	tlsConfig := tls.Config{
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			misconfiguredCertCN := auditChain(rawCerts)
			if misconfiguredCertCN != "" {
				fmt.Println(misconfiguredCertCN)
			}
			return nil
		},
	}
	tls.DialWithDialer(&dialer, "tcp", fmt.Sprintf("%s:443", hostname), &tlsConfig)
	return
}

// reverseHostname for a given hostname reverses the hostname from the
// stats-exporter hostname format: <tld label> followed by each <label>
// of the fqdn back to a proper fqdn
func reverseHostname(hostname string) string {
	labels := strings.Split(hostname, ".")
	for i, j := 0, len(labels)-1; i < j; i, j = i+1, j-1 {
		labels[i], labels[j] = labels[j], labels[i]
	}
	return strings.Join(labels, ".")
}

// statsCsvToHostnames expects a csv file path produced by
// stats-exporter in the sre-tools repo, parses it, reverses the
// hostname entry from the first column of each row (back) into a proper
// fqdn and appends it to a slice of strings
func statsCsvToHostnames(statsCsv string) []string {
	csvFile, err := os.Open(statsCsv)
	if err != nil {
		log.Fatalln("Couldn't open the csv file", err)
	}

	hostnames := []string{}
	r := csv.NewReader(csvFile)
	for {
		entry, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalln("Issue parsing entry in csv file", err)
		}
		hostnames = append(hostnames, reverseHostname(entry[0]))
	}
	return hostnames
}

func main() {
	flag.StringVar(&statsCsv, "stats-csv-file", "", "path to csv file produced by stats-exporter")
	flag.BoolVar(&debugMode, "debug", false, "Print full audit output for every hostname")
	flag.Parse()
	var hostnames []string
	if statsCsv != "" {
		hostnames = statsCsvToHostnames(statsCsv)
	} else {
		hostnames = os.Args[1:]
	}

	if len(hostnames) == 0 {
		fmt.Print("You must supply at least one hostname via stdin or csv file using `--stats-exporter-file`")
		os.Exit(1)
	}

	c := make(chan string, len(hostnames))

	go func() {
		for _, hostname := range hostnames {
			c <- hostname
		}
	}()

	for hostname := range c {
		wg.Add(1)
		go auditHostname(hostname)
		wg.Wait()
	}
	close(c)
	fmt.Print("Done")
}
