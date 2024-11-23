#!/bin/sh
set -e

CHROOT_DIR=$(mktemp -d)
initramfs="$PWD"/initramfs-lts
cd "$CHROOT_DIR"
gunzip -c "$initramfs" | cpio -i
mount -t proc none proc
mount -o bind /sys /sys
chroot . /bin/sh -i
