#!/bin/bash

# Go to hell cloud uploads
export ASCIINEMA_API_URL='http://127.0.0.1'

for X in testguy; do
    if [ -d /tmp/tmux-$(id -u ${X}) ]; then
        # Get list of shared tmux sessions
        for P in $(find /tmp/tmux-$(id -u ${X}) -type s | awk -F'-' '{print $3}'); do

            # Check if the tmux session corresponding to the shared socket name is still active.
            # Clean up if it's not.
            ps --no-headers --pid ${P} > /dev/null 2>&1
            if [ $? -ne 0 ]; then
                echo "Cleaning up /tmp/tmux-$(id -u ${X})/${X}-${P} because it is stale"
                rm -f "/tmp/tmux-$(id -u ${X})/${X}-${P}"
            fi

            # If there is an active tmux socket(s) for the monitored user, we'll spawn a single
            # asciinema recording session inside the monitored users tmux, but we'll be read-only.
            if [ -S "/tmp/tmux-$(id -u ${X})/${X}-${P}" ]; then
                ps aux | grep asciinema | grep /tmp/tmux-$(id -u ${X})/${X}-${P} > /dev/null 2>&1
                if [ $? -ne 0 ]; then
                    asciinema rec --quiet --command "tmux -S /tmp/tmux-$(id -u ${X})/${X}-${P} attach -t ${X}-${P} -r"
                fi
            fi

        done
    fi
done
