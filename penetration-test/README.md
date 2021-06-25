# Penetration Test monitoring tools

We are required to monitor the penetration testers activity when they are connected to our CA systems via the VPN tunnel. We'll employ the tooling in this repository to help us do so.

## Setup

```
scp -r sre-tools/penetration-test pporada@${PENTEST_JUMP_BOX_IP}:

ssh pporada@${PENTEST_JUMP_BOX_IP}

sudo apt update
sudo apt install -y socat asciinema inotify-tools

sudo cp penetration-test/tmux-login-shell-wrapper.sh /bin/
sudo chsh -s /bin/tmux-login-shell-wrapper.sh ${PENTEST_USER}
sudo cp penetration-test/tmux.conf /home/${PENTEST_USER}/.tmux.conf

sudo cp penetration-test/tmux-admin-wrapper.sh /bin
sudo chown root:sres /bin/tmux-admin-wrapper.sh
sudo chmod 750 /bin/tmux-admin-wrapper.sh

sudo cp penetration-test/detect-login.sh /bin
sudo chown root:sres /bin/detect-login.sh
sudo chmod 750 /bin/detect-login.sh

sudo mkdir -p /tmp/asciinema-recordings
sudo chown root:pentest /tmp/asciinema-recordings
sudo chmod 775 /tmp/asciinema-recordings
```

## How to use this

### SRE
When you sign in to monitor the pentester's, run the following:

```
ps aux | grep detect-login.sh | grep -v grep
```

If there isn't a hit, spawn that script in a screen or something so it stays running.
```
screen
/bin/detect-login.sh
# Detach from that screen session, you won't need it anymore
```

Next, create/attach to a shared SRE tmux session that allows each SRE to independently navigate tmux:
```
/bin/tmux-admin-wrapper.sh
```

Screen output logs from the pentester's session can be found at `/tmp/asciinema-recordings/*.txt`.

### Pentester
All a pentester has to do is login to the server. Their default shell will be set to `/bin/tmux-login-shell-wrapper.sh` which spawns a tmux session with a socket a known location. Upon creation of that socket, SRE tooling will automatically begin recording and monitoring the pentester tmux session.
