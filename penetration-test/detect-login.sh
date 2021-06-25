#!/bin/bash
EXCLUDE_REGEX='(.git/|.lock|default|.*(\.sw.|~)|pporada|jrenken|aomidi|root|user)'

# The export is required, otherwise asciinema will still try to upload the recording to their servers
export ASCIINEMA_API_URL='http://127.0.0.1'

inotifywait -qmr --format '%w%f' --exclude "${EXCLUDE_REGEX}" -e create /tmp/tmux* | while read FILE; do
    echo "[$(date --rfc-3339=seconds)] Tmux session detected: ${FILE}"
    PENTEST_TMUX_SOCKET=${FILE}
    PENTEST_TMUX_SESSION=${PENTEST_TMUX_SOCKET##*/}
    PENTEST_TMUX_USER=$(echo ${PENTEST_TMUX_SESSION} | awk -F'-' '{print $2}')
    PENTEST_SHARED_GROUP="qubes"
    SRE_TMUX_SESSION="sre-shared-session"

    # Record the pentester's shell session
    # TODO: Upload the .txt files to S3 or something
    # TODO: Do we even need asciinema? Will script suffice?
    asciinema rec --quiet --command "tmux -S ${PENTEST_TMUX_SOCKET} attach -t ${PENTEST_TMUX_SESSION} -r" > /dev/null 2>&1 | tee -a /tmp/asciinema-recordings/${PENTEST_TMUX_SESSION}.txt | socat STDIO UNIX-LISTEN:/tmp/asciinema-recordings/${PENTEST_TMUX_SESSION}.cast,user=${PENTEST_TMUX_USER},group=${PENTEST_SHARED_GROUP},mode=775 &

    # Allow SREs to view the socat output coming from the pentester's shell
    # When the pentester closes their terminal session, this backgrounded tmux session's window will close and clean up after itself
    tmux -S /tmp/${SRE_TMUX_SESSION} new-window -n ${PENTEST_TMUX_SESSION} -t ${SRE_TMUX_SESSION}: "socat UNIX-CONNECT:/tmp/asciinema-recordings/${PENTEST_TMUX_SESSION}.cast STDOUT" & 
done
