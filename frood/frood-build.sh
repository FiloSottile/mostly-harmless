#!/bin/sh
set -e

__() { printf "\n\033[1;32m* %s [%s]\033[0m\n" "$1" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"; }

__ "Fetching alpine-make-rootfs"

wget https://raw.githubusercontent.com/alpinelinux/alpine-make-rootfs/v0.7.0/alpine-make-rootfs \
    && echo 'e09b623054d06ea389f3a901fd85e64aa154ab3a  alpine-make-rootfs' | sha1sum -c && \
    chmod +x alpine-make-rootfs

ROOTFS_DEST=$(mktemp -d)
IMAGE_DEST="/mnt/images/$1"
rm -rf "$IMAGE_DEST"

__ "Building Go binaries"

apk add --no-cache go
go env -w GOTOOLCHAIN=auto
go build -C /mnt -o "$ROOTFS_DEST/usr/local/bin/" ./bins/...

__ "Building rootfs"

mkdir -p "$ROOTFS_DEST/etc"
echo "$1" > "$ROOTFS_DEST/etc/frood-release"

# Stop mkinitfs from running during apk install.
mkdir -p "$ROOTFS_DEST/etc/mkinitfs"
echo "disable_trigger=yes" > "$ROOTFS_DEST/etc/mkinitfs/mkinitfs.conf"

# Stop update-extlinux from running during apk install.
echo "disable_trigger=1" > "$ROOTFS_DEST/etc/update-extlinux.conf"

export ALPINE_BRANCH=edge
export SCRIPT_CHROOT=yes
export FS_SKEL_DIR=/mnt/root
export FS_SKEL_CHOWN=root:root
PACKAGES="$(grep -v -e '^#' -e '^$' /mnt/packages)"
export PACKAGES
./alpine-make-rootfs "$ROOTFS_DEST" /mnt/setup.sh

__ "Building initramfs"

cd "$ROOTFS_DEST"
mv boot "$IMAGE_DEST"
find . | cpio -o -H newc | gzip > "$IMAGE_DEST/initramfs-lts"

__ "Created image $1!"

du -hs .
