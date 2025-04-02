#!/bin/sh
set -e

__() { printf "\n\033[1;32m* %s [%s]\033[0m\n" "$1" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"; }

ROOTFS_DEST=$(mktemp -d)

__ "Fetching alpine-make-rootfs"

wget https://raw.githubusercontent.com/alpinelinux/alpine-make-rootfs/v0.7.0/alpine-make-rootfs \
    && echo '91ceb95b020260832417b01e45ce02c3a250c4527835d1bdf486bf44f80287dc  alpine-make-rootfs' \
    | sha256sum -c || exit 1 && chmod +x alpine-make-rootfs

__ "Building Go binaries"

export GOTOOLCHAIN=auto
go build -C /mnt -o "$ROOTFS_DEST/usr/local/bin/" ./bins/...

__ "Building rootfs"

mkdir -p "$ROOTFS_DEST/etc"
basename "$1" > "$ROOTFS_DEST/etc/frood-release"

# Stop mkinitfs from running during apk install.
mkdir -p "$ROOTFS_DEST/etc/mkinitfs"
echo "disable_trigger=yes" > "$ROOTFS_DEST/etc/mkinitfs/mkinitfs.conf"

export ALPINE_BRANCH=edge
export SCRIPT_CHROOT=yes
export FS_SKEL_DIR=/mnt/root
export FS_SKEL_CHOWN=root:root
PACKAGES="$(grep -v -e '^#' -e '^$' /mnt/packages)"
export PACKAGES
./alpine-make-rootfs "$ROOTFS_DEST" /mnt/setup.sh

__ "Building initramfs"

cd "$ROOTFS_DEST"
find . -path "./boot" -prune -o -print | cpio -o -H newc | gzip > "$ROOTFS_DEST/boot/initramfs-lts"

__ "Building UKI image"

apk add --no-cache --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing/ \
    systemd-efistub ukify # https://gitlab.alpinelinux.org/alpine/aports/-/issues/16691

# The default rdinit is /init, while the default init is /sbin/init.
CMDLINE="rdinit=/sbin/init console=tty1 console=ttyAMA0"

ukify build --output "$1.efi" --cmdline "$CMDLINE" \
    --linux "$ROOTFS_DEST/boot/vmlinuz-lts" \
    --initrd "$ROOTFS_DEST/boot/initramfs-lts" \
    --os-release "@$ROOTFS_DEST/etc/frood-release"

__ "Building ESP image"

apk add --no-cache sfdisk mtools
# https://unix.stackexchange.com/a/527217/323803
truncate -s $((256*1024*1024)) "$1.img"
printf "label: gpt\ntype=uefi" | sfdisk "$1.img"
FS="$1.img"@@$((1024*1024))
mformat -i "$FS" -t 254 -h 64 -s 32 -v frood
mmd -i "$FS" ::EFI
mmd -i "$FS" ::EFI/BOOT
mcopy -i "$FS" "$1.efi" ::EFI/BOOT/BOOTAA64.EFI

__ "Created image!"

ls -lh "$1.efi" "$1.img"
