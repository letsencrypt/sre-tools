# chain-auditor
For a given list of hostnames, this tool dials and starts a TLS handshake and
audits the presented (raw) certificate chain for inconsistencies between the
Issuer Subject Name of the leaf certificate and the Subject Name of the of the
intermediate certificate.

## Inputs
Acceptable inputs are, space separated hostnames supplied as arguments or a path
to a TSV (tab separated value) file provided following the `--stats-tsv-file`
flag.

### TSV Format
The TSV file must contain the hostname in the second column of every row and the
hostname must be in the following format `<tld label>` followed by each
`<label>`of the hostname in reverse order (i.e. mail.google.com would be
com.google.mail)

## Usage
With hostnames provided via `--stats-tsv-file` and `parallelism` (number of
concurrent hostnames to dial) set to `2`:
```shell
$ $ go run ./main.go --stats-tsv-file test_hostnames.tsv --parallelism 2
0.8% (█                    ) audit/s(2.8) mismatches(0) unreachable(2) remain(894) dns(0) netTimeout(2) netOther(0)
```

## Current Setup
This job is currently running on the `chain-auditor` AWS instance. The stats
files required for this job created by a database job in our infra and
transmitted via `scp` to the `cthulk` AWS instance.

On the `chain-auditor` instance, the cron that runs `daily-work.sh` is owned and
run by the `chain-audit` user. This user has the same UID as the `ec2-user` on
`cthulk` which allows them the same permissions to the files residing in
`/mnt/efs/`, an AWS EFS filesystem mounted as an NFS share.

Daily, when new gzipped stats files land in the EFS share, `daily-work.sh`
unpacks the `tsv` file to the `chain-audit` homedir, runs `chain-auditor` to
produce a results file, deletes the unpacked stats file, and moves the results
file into the `audit-results` dir.
