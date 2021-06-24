#!/bin/bash

# Don't tmux inside tmux
if [ -n ${TMUX} ]; then
    DATE=$(date +%Y%m%d%S)
    SESSION_NAME="${DATE}-${USER}-${BASHPID}"
    # Go to hell cloud uploads
    export ASCIINEMA_API_URL='http://127.0.0.1'

    # On the pentest jump box, we'll want this folder to have a shared group for the pentesters
    ## chown root:qubes /tmp/asciinema-recordings
    ## chmod 775 /tmp/asciinema-recordings
    mkfifo /tmp/asciinema-recordings/${SESSION_NAME}.cast

    # We are trusting that the pentesters don't kill the recording session
    # We're having the pentesters login shell create recording session so that
    # tmux capturing works
    nohup asciinema rec /tmp/asciinema-recordings/${SESSION_NAME}.cast --quiet > /dev/null 2>&1 &

    # We're going to create a shared session, but without the actual sharing. This
    # allows us to get a socket whose name we can parse with the admin monitor tool.
    # The socket isn't automatically cleaned up by tmux upon exit.
    tmux -L ${SESSION_NAME} new-session -d -s ${SESSION_NAME}
    tmux -L ${SESSION_NAME} select-window -t ${SESSION_NAME}:0

    # exec will prevent the script from exiting after it spawns tmux thereby allowing you to "use" tmux
    exec tmux -L ${SESSION_NAME} attach-session -t ${SESSION_NAME}

    # Set our variables
    tmux -L ${SESSION_NAME} source-file /etc/tmux.conf
else
    exit 0
fi
