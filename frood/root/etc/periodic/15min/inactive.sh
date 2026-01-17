#!/bin/sh
set -eu

if [ "$(cut -d' ' -f1 /proc/uptime | cut -d'.' -f1)" -lt 3600 ]; then
    exit 0
fi

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
