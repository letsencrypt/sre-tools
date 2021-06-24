#!/bin/bash
EXCLUDE_REGEX='(.git/|.lock|default|.*(\.sw.|~)|pporada|jrenken|aomidi|root)'

inotifywait -qmr --format '%w%f' --exclude "${EXCLUDE_REGEX}" -e create /tmp/tmux* | while read FILE; do
    echo "[$(date --rfc-3339=seconds)] Tmux session detected: ${FILE}"
    nohup ./monitor-new.sh ${FILE} > /dev/null 2>&1 &
done
