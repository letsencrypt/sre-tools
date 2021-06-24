#!/bin/bash
# We're going to create a shared socket, but each admin will have their
# own session created by running this script. The methodology is to create
# a window group that allows multiple sessions to connect to the same
# tmux server, but each tmux client can view independant tmux windows.

# Don't tmux inside tmux
if [ -n ${TMUX} ]; then
    SESSION_NAME="sre-shared-session"

    # The socket isn't automatically cleaned up by tmux upon exit.
    if [ ! -S /tmp/${SESSION_NAME} ]; then
        tmux -S /tmp/${SESSION_NAME} new-session -d -s ${SESSION_NAME}
        chown root:qubes /tmp/${SESSION_NAME}

        # Set our variables
        tmux -S /tmp/${SESSION_NAME} source-file /etc/tmux.conf
    fi

    # Attach to the window group of the shared session
    tmux -S /tmp/${SESSION_NAME} new-session -A -t ${SESSION_NAME}
else
    echo "Already inside tmux, exiting."
    exit 0
fi

