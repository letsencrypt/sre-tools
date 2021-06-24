#!/bin/bash
EXCLUDE_REGEX='(.git/|.lock|default|.*(\.sw.|~))'

inotifywait -qmr --format '%w%f' --exclude "${EXCLUDE_REGEX}" -e create /tmp/asciinema-recordings | while read FILE; do
    #nohup ./monitor-new.sh ${FILE} > /dev/null 2>&1 &
    echo ${FILE}
done

function wip() {
    SESSION_NAME="sre-shared-session"
    tmux -S /tmp/${SESSION_NAME} attach-session -t ${SESSION_NAME}
}
