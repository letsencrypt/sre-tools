# chain-auditor
For a given list of hostnames, this tool dials and starts a TLS
handshake and audits the presented (raw) certificate chain for
inconsistencies between the Issuer Subject Name of the leaf certificate
and the Subject Name of the of the intermediate certificate.

## Inputs
Acceptable inputs are, space separated hostnames supplied as arguments
or a path to a TSV (tab separated value) file provided following the
`--stats-tsv-file` flag.

### TSV Format
The TSV file must contain the hostname in the second column of every row
and the hostname must be in the following format `<tld label>` followed
by each `<label>`of the hostname in reverse order (i.e. mail.google.com
would be com.google.mail)

## Debug Mode
Provided also is a debug mode. If the operator supplies the flag --debug
all hostnames audited will have their results printed in the following
format: 

```
leafCert: [subjectCN: <subjectCN> | issuerCN: <issuerCN>] -> chainCert0: [subjectCN: <subjectCN> | issuerCN: <issuerCN>]
```

## Usage

With debug flag and hostname provided as an argument:
```shell
$ go run ./cmd/chain-auditor/main.go --debug letsencrypt.org
leafCert: [subjectCN: lencr.org | issuerCN: Let's Encrypt Authority X3] -> chainCert0: [subjectCN: Let's Encrypt Authority X3 | issuerCN: DST Root CA X3]
```

With debug flag and hostnames provided via `--stats-tsv-file`:
```shell
$ go run ./cmd/chain-auditor/main.go --debug --stats-tsv-file cmd/chain-auditor/test_hostnames.tsv
leafCert: [subjectCN: migom.com | issuerCN: Cloudflare Inc ECC CA-3] -> chainCert0: [subjectCN: Cloudflare Inc ECC CA-3 | issuerCN: Baltimore CyberTrust Root]
leafCert: [subjectCN: sni.cloudflaressl.com | issuerCN: Cloudflare Inc ECC CA-3] -> chainCert0: [subjectCN: Cloudflare Inc ECC CA-3 | issuerCN: Baltimore CyberTrust Root]
leafCert: [subjectCN: saltstack.com | issuerCN: Amazon] -> chainCert0: [subjectCN: Amazon | issuerCN: Amazon Root CA 1]  -> chainCert1: [subjectCN: Amazon Root CA 1 | issuerCN: Starfield Services Root Certificate Authority - G2]  -> chainCert2: [subjectCN: Starfield Services Root Certificate Authority - G2 | issuerCN: ]
leafCert: [subjectCN: tjprc.org | issuerCN: cPanel, Inc. Certification Authority] -> chainCert0: [subjectCN: cPanel, Inc. Certification Authority | issuerCN: COMODO RSA Certification Authority]  -> chainCert1: [subjectCN: COMODO RSA Certification Authority | issuerCN: AAA Certificate Services]
...
```