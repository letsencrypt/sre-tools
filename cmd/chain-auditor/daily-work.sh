#!/bin/env bash
set -eu

# Mount point of the EFS volume
MOUNT_POINT="/mnt/efs"

# Filename of the daily stats tsv file, this is gzipped
STATS_FILENAME="$(date +"results-%Y-%m-$(expr $(date +%d) - 2).tsv.gz")"

# Filename of the unpacked stats tsv file
UNPACKED_STATS_FILENAME="${STATS_FILENAME%.*}"

# Expected name of the chain-auditor results file, should always be
# `chain-auditor-<STATS_FILE>`
RESULTS_FILENAME="chain-audit-"${STATS_FILENAME%.*}""

# Unpack the gzipped stats tsv to our homedir
zcat "${MOUNT_POINT}"/"${STATS_FILENAME}">"${UNPACKED_STATS_FILENAME}"

# Run the actual audit job against the unpacked stats tsv file
./chain-auditor --stats-tsv-file "${UNPACKED_STATS_FILENAME}" --parallelism 150

# Move the finished audit log to a folder with all of it's friends, also makes
# it easier to rsync or scp from this host
mv "${RESULTS_FILENAME}" chain-audit-results/

# Delete the unpacked stats tsv file, our homedir only has about 8GB of free
# space, we want to keep it clear
rm "${UNPACKED_STATS_FILENAME}"
