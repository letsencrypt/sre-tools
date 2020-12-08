package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/superhawk610/bar"
)

const (
	r3 = "R3"
)

var debugMode bool

type probs struct {
	netErrTimeout bool
	netErrOther   bool
	dnsErr        bool
}

type result struct {
	hostname   string
	reachable  bool
	tls        string
	mismatched bool
	ip         string
	agent      string
	probs      probs
}

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

// certBytesToChain marshals a slice of byte slices representing an x.509
// certificate chain to a slice of *x.509Certificate objects
func certBytesToChain(rawCerts [][]byte) []*x509.Certificate {
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

// mismatchInChain for a given slice of byte slices representing an
// x.509 certificate chain, if the Issuer Common Name is const r3,
// validates that the resulting chain of x509 Certificates contains the
// corresponding r3 intermediate that issued the leaf Certificate.
func mismatchInChain(rawCerts [][]byte) bool {
	chain := certBytesToChain(rawCerts)
	leafIssuerCN := chain[0].Issuer.CommonName
	if len(chain) > 1 {
		if leafIssuerCN == r3 && !chainContainsR3(chain) {
			return true
		}
	}
	return false
}

// getConnectProbs for a given error resulting from an attempt to tcp
// dial a hostname of port 443 classify the error as a DNS lookup, Dial
// Timeout, or Network Other
func getConnectProbs(err error) probs {
	probs := probs{}
	var dnsErr *net.DNSError
	var netErr net.Error

	if errors.As(err, &dnsErr) {
		probs.dnsErr = true
	}

	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			probs.netErrTimeout = true
		} else if !probs.dnsErr {
			probs.netErrOther = true
		}
	}
	return probs
}

func parseIPFromConnection(conn *tls.Conn) string {
	return net.ParseIP(
		strings.ReplaceAll(
			conn.RemoteAddr().String(), ":443", "")).String()
}

func parseTLSVersionFromConn(conn *tls.Conn) string {
	tlsVersions := map[uint16]string{
		tls.VersionTLS10: "1.0",
		tls.VersionTLS11: "1.1",
		tls.VersionTLS12: "1.2",
		tls.VersionTLS13: "1.3",
	}
	return tlsVersions[conn.ConnectionState().Version]
}

// auditChainForHostname for a given hostname, dials and starts a TLS handshake.
// The tls.Config skips verification steps and delegates verification to
// an anonymous function that audits the certification chain
func auditChainForHostname(hostname string) result {
	var mismatched bool
	dialer := net.Dialer{Timeout: 1 * time.Second}
	tlsConfig := tls.Config{
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			mismatched = mismatchInChain(rawCerts)
			return nil
		},
	}
	conn, err := tls.DialWithDialer(&dialer, "tcp", fmt.Sprintf("%s:443", hostname), &tlsConfig)
	result := result{hostname: hostname}
	if err != nil {
		result.probs = getConnectProbs(err)
		result.mismatched = mismatched
		return result
	}
	defer conn.Close()
	result.mismatched = mismatched
	result.tls = parseTLSVersionFromConn(conn)
	result.ip = parseIPFromConnection(conn)
	result.reachable = true
	return result
}

// setupProgressBar sets the format string used when the progress bar is
// running and the column width the bar takes up
func setupProgressBar(total int) *bar.Bar {
	progressBar := bar.NewWithOpts(
		bar.WithDimensions(total, 20),
		bar.WithFormat(
			":percent :bar audit/s(:rate) mismatches(:mismatched) unreachable(:unreachable) remain(:remain) dns(:dns) netTimeout(:timeout) netOther(:other) "),
	)

	return progressBar
}

// shuffleHostnames shuffles hostnames to reduce concurrent calls to the
// same origin
func shuffleHostnames(hostnames []string) []string {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(hostnames), func(i, j int) { hostnames[i], hostnames[j] = hostnames[j], hostnames[i] })
	return hostnames
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

// statsTsvToHostnames expects a tsv file path produced by
// stats-exporter in the sre-tools repo, parses it, reverses the
// hostname entry from the second column of each row (back) into a proper
// fqdn and appends it to a slice of strings
func statsTsvToHostnames(statsTsv string) []string {
	tsvFile, err := os.Open(statsTsv)
	if err != nil {
		log.Fatalln("Couldn't open the tsv file", err)
	}
	hostnames := []string{}
	r := csv.NewReader(tsvFile)
	r.Comma = '\t'
	for {
		entry, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalln("Issue parsing entry in tsv file", err)
		}
		// *.example.com will not resolve, we shouldn't try, this one
		// line reduces our hostnames list by ~10%
		if strings.Contains(entry[1], "*") {
			continue
		}
		hostnames = append(hostnames, reverseHostname(entry[1]))
	}
	return hostnames
}

func getHostnames(statsTsv *string) []string {
	var hostnames []string
	hostnames = statsTsvToHostnames(*statsTsv)
	if len(hostnames) == 0 {
		fmt.Print("You must supply at least one hostname as an argument or a file of using `--stats-tsv-file`")
		os.Exit(1)
	}
	return shuffleHostnames(hostnames)

}

func parseCLIOptions() (*string, *int) {
	flag.BoolVar(&debugMode, "debug", false, "Print full audit output for every hostname with a mismatched intermediate")
	statsTsv := flag.String("stats-tsv-file", "", "path to tab separated value file produced by stats-exporter")
	parallelism := flag.Int("parallelism", 1, "Specify the number of co-routines to use")
	flag.Parse()
	return statsTsv, parallelism
}

func main() {
	statsTsv, parallelism := parseCLIOptions()
	hostnames := getHostnames(statsTsv)

	fileName := fmt.Sprintf("chain-audit-%s", time.Now().Format("2006-01-02"))
	if *statsTsv != "" {
		fileName = fmt.Sprintf("chain-audit-%s", *statsTsv)
	}

	auditFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	progressBar := setupProgressBar(len(hostnames))
	hnChan := make(chan string, len(hostnames))
	resChan := make(chan result)
	doneChan := make(chan bool, 1)
	hostnamesRemainCount := len(hostnames)

	var dnsCount int
	var timeoutCount int
	var otherCount int
	var unreachableCount int
	var mismatchedCount int

	go func() {
		for _, hostname := range hostnames {
			hnChan <- hostname
		}
		close(hnChan)
	}()

	var wg sync.WaitGroup
	for i := 0; i < *parallelism; i++ {
		wg.Add(1)
		go func() {
			for hostname := range hnChan {
				result := auditChainForHostname(hostname)
				if result.mismatched == true {
					if debugMode == true {
						fmt.Printf("%+v\n", result)
					}
					resChan <- result
					mismatchedCount++
				}
				hostnamesRemainCount--
				if !result.reachable == true {
					unreachableCount++
					if result.probs.dnsErr == true {
						dnsCount++
					}
					if result.probs.netErrTimeout == true {
						timeoutCount++
					}
					if result.probs.netErrOther == true {
						otherCount++
					}
				}
				progressBar.TickAndUpdate(bar.Context{
					bar.Ctx("mismatched", strconv.Itoa(mismatchedCount)),
					bar.Ctx("remain", strconv.Itoa(hostnamesRemainCount)),
					bar.Ctx("unreachable", strconv.Itoa(unreachableCount)),
					bar.Ctx("dns", strconv.Itoa(dnsCount)),
					bar.Ctx("timeout", strconv.Itoa(timeoutCount)),
					bar.Ctx("other", strconv.Itoa(otherCount)),
				})
			}
			wg.Done()
		}()
	}

	go func() {
		for result := range resChan {
			_, err := auditFile.Write([]byte(fmt.Sprintf("%s\t%s\n", result.hostname, result.ip)))
			if err != nil {
				log.Fatal(err)
			}
		}
		doneChan <- true
	}()
	wg.Wait()
	progressBar.Done()
	close(resChan)
	<-doneChan

	_, err = auditFile.Write([]byte(
		fmt.Sprintf(
			"total:%d\tmismatched:%d\tunreachable:%d\terrdns:%d\terrtimeout:%d\terrnetother:%d", len(hostnames), mismatchedCount,
			unreachableCount, dnsCount, timeoutCount, otherCount)))
	if err != nil {
		log.Fatal(err)
	}
	if err := auditFile.Close(); err != nil {
		log.Fatal(err)
	}
}
