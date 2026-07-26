#!/bin/sh

. /etc/conf.d/iliad

cgroup="/sys/fs/cgroup/$table"

print_and_run() {
    echo
    echo "$ $*"
    "$@"
}

print_and_run ip -4 addr show dev "$iface"
print_and_run ip -4 route show table main
print_and_run ip -4 route show table "$table"
print_and_run ip rule list
print_and_run nft list table ip "$table"
print_and_run cat "$cgroup/cgroup.procs"
