# Penetration Test monitoring tools

We are required to monitor the penetration testers' activity when they are connected to our CA systems via the VPN tunnel. We'll employ the tooling in this repository to help us do so.

## Setup

On your VPN qube:

```
PENTEST_JUMP_BOX_IP="1.1.1.1"

sshuser=$(git config --get user.email | awk -F'@' '{ print $1 }')

scp -r sre-tools/penetration-test ${sshuser}@${PENTEST_JUMP_BOX_IP}:

ssh ${sshuser}@${PENTEST_JUMP_BOX_IP}
```

On the pentest jump host:

```
# Change this to the list of penetration testers, if their accounts already
# exist. Otherwise, make it an empty string.
GEESE="duck goose greyduck"

sudo apt update
sudo apt --assume-yes install \
  asciinema inotify-tools members socat tmux

sudo install                           \
  --owner root --group sre --mode 0644 \
  penetration-test/tmux.conf           \
  /etc/skel/.tmux.conf

for GOOSE in ${GEESE};
do
    sudo install                                   \
      --owner ${GOOSE} --group pentest --mode 0644 \
      /etc/skel/.tmux.conf                         \
      /home/${GOOSE}/
done

sudo install                                \
  --owner root --group sre --mode 0755      \
  penetration-test/tmux-login-shell-wrapper \
  /usr/local/bin/

for GOOSE in ${GEESE};
do
    sudo chsh -s /usr/local/bin/tmux-login-shell-wrapper ${GOOSE}
done

sudo install                                \
  --owner root --group sre --mode 0750      \
  penetration-test/detect-login             \
  penetration-test/tmux-admin-wrapper       \
  /usr/local/sbin/

sudo mkdir -p /tmp/asciinema-recordings
sudo chown root:pentest /tmp/asciinema-recordings
sudo chmod 0775 /tmp/asciinema-recordings
```

## How to use this

### SRE
When you sign in to monitor the pentesters, run the following:

```
ps aux | grep detect-login | grep -v grep
```

If there isn't a hit, spawn that script in a screen or something so it stays running:
```
screen -d -m detect-login
# Leave that screen session detached; you won't need it any more.
```

Next, create/attach to a shared SRE tmux session that allows each SRE to independently navigate tmux:
```
tmux-admin-wrapper
```

Screen output logs from the pentesters' sessions can be found at `/tmp/asciinema-recordings/*.txt`.

### Pentester
All a pentester has to do is log into the server. Their default shell will be set to `tmux-login-shell-wrapper`, which spawns a `tmux` session with a socket at a known location. Upon that socket's creation, SRE tooling will automatically begin recording and monitoring the pentester's `tmux` session.
