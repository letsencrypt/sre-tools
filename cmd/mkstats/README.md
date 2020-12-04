These are tools to calculate stats for https://letsencrypt.org/stats.

Given a TSV dump of the issuedNames table from a running Boulder instance, you
can pass it to `splitter` to split the lines across files named, e.g.
`2018-01-02.tsv`, where each file contains only entries on that particular date.

After running splitter, run `active -startDate XXX -endDate YYY -outFile out.tsv` 
(endDate is exclusive). With the `-outFile` flag, dates that already exist in
the output will be skipped, so it's fine to run multiple times.

Since `active` loads all entries for a 90-day period into memory, it needs about
12 GB of memory to run on the latest data, as of January 2018.
