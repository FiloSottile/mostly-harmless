#!/bin/sh

# Thanks to https://github.com/jpetazzo/dockvpn for the tun/tap fix
mkdir -p /dev/net
mknod /dev/net/tun c 10 200

exec iodined -c -f 10.16.0.1 $IODINE_HOST -P $IODINE_PASSWORD >>/var/log/iodined.log 2>&1
