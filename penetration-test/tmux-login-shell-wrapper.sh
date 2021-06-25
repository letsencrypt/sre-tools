#!/bin/bash

# Don't tmux inside tmux
if [ -n ${TMUX} ]; then
    SESSION_NAME="${DATE}-${USER}-${BASHPID}"
    tmux -L ${SESSION_NAME} new-session -d -s ${SESSION_NAME}
    tmux -L ${SESSION_NAME} select-window -t ${SESSION_NAME}:0
    # exec will prevent the script from exiting after it spawns tmux thereby allowing you to "use" tmux
    exec tmux -L ${SESSION_NAME} new-session -A -t ${SESSION_NAME}
fi
