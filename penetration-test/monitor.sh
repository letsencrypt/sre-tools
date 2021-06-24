#!/bin/bash

# Go to hell cloud uploads
export ASCIINEMA_API_URL='http://127.0.0.1'

# This comes from ./inotify.sh
TMUX_SOCKET=${1}
TMUX_SESSION=${TMUX_SOCKET##*/}
echo ${TMUX_SESSION}

# TODO: Find a better place to store these. Ideally we should send them up to S3 after we close the VPN tunnel.
DATE=$(date +%Y%m%d)
if [ ! -d "/tmp/asciinema-recordings/${DATE}" ]; then
    mkdir -p "/tmp/asciinema-recordings/${DATE}"
fi

asciinema rec /tmp/asciinema-recordings/${DATE}/${TMUX_SESSION}.cast \
    --quiet \
    --command "tmux -S ${TMUX_SOCKET} attach -t ${TMUX_SESSION} -r"
