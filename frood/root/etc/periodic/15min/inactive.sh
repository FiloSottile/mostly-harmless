#!/bin/sh
set -eu

for name in sshd-session tmux ksmbd:; do
    if pgrep "$name" >/dev/null 2>&1; then
        exit 0
    fi
done

if zpool status | grep -q "in progress"; then
    exit 0
fi

pstree | /usr/local/bin/mail-alert "frood inactivity power-off"

poweroff -d 120
