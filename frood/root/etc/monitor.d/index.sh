#!/bin/sh

cat /etc/frood-release
cat /etc/motd

uptime
uname -a

print_and_run() {
    echo
    echo "$ $*"
    "$@"
}

print_and_run zpool status
print_and_run df -h
print_and_run free -h
print_and_run pstree
print_and_run rc-status -a
print_and_run ip addr
print_and_run tailscale status

print_and_run ls /etc/monitor.d
