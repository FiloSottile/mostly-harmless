#!/bin/sh
set -e

apk add --no-cache sfdisk mtools

# https://unix.stackexchange.com/a/527217/323803
truncate -s $((1024*1024*1024)) "$2"
printf "label: gpt\ntype=uefi" | sfdisk "$2"
FS="$2"@@$((1024*1024))
mformat -i "$FS" -F -t 254 -h 64 -s 32 -v frood
printf '\xE7\x19\x1B\xB6' | dd of="$2" bs=1 count=4 seek=$(( (1024*1024) + 67 )) conv=notrunc
mmd -i "$FS" ::EFI
mmd -i "$FS" ::EFI/BOOT
mcopy -i "$FS" "$1" ::EFI/BOOT/BOOTAA64.EFI
