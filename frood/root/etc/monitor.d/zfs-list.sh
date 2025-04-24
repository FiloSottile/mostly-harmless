#!/bin/sh
zfs list -o name,used,avail,refer,mountpoint,canmount,readonly,compression,encryption
