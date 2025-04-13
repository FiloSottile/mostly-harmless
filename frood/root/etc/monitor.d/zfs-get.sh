#!/bin/sh
zfs get -s local,temporary,received all
echo
zpool get all
