#!/bin/sh
set -e

wget https://raw.githubusercontent.com/alpinelinux/alpine-make-rootfs/v0.7.0/alpine-make-rootfs \
    && echo 'e09b623054d06ea389f3a901fd85e64aa154ab3a  alpine-make-rootfs' | sha1sum -c && \
    chmod +x alpine-make-rootfs

ROOTFS_DEST=$(mktemp -d)
IMAGE_DEST="/mnt/images/$1"
rm -rf "$IMAGE_DEST"

mkdir -p "$ROOTFS_DEST/etc"
echo "$1" > "$ROOTFS_DEST/etc/frood-release"

# Stop mkinitfs from running during apk install.
mkdir -p "$ROOTFS_DEST/etc/mkinitfs"
echo "disable_trigger=yes" > "$ROOTFS_DEST/etc/mkinitfs/mkinitfs.conf"

# syslinux apk trigger WARNING is expected.
# https://gitlab.alpinelinux.org/alpine/aports/-/issues/16560

export ALPINE_BRANCH=edge
export SCRIPT_CHROOT=yes
export FS_SKEL_DIR=/mnt/root
export FS_SKEL_CHOWN=root:root
PACKAGES="$(cat /mnt/packages)"
export PACKAGES
./alpine-make-rootfs "$ROOTFS_DEST" /mnt/setup.sh

cd "$ROOTFS_DEST"
mv boot "$IMAGE_DEST"
find . | cpio -o -H newc | gzip > "$IMAGE_DEST/initramfs-lts"

echo "Created image $1!"
du -hs .
