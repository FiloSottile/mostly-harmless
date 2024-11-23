#!/bin/sh
set -ex

rm -rf /media/usb/boot/syslinux
mkdir -p /media/usb/boot/syslinux

cp /usr/share/syslinux/*.c32 /media/usb/boot/syslinux/

extlinux --install /media/usb/boot/syslinux

cat > /media/usb/boot/syslinux/syslinux.cfg <<EOF
PROMPT 0
DEFAULT lts

LABEL lts
KERNEL /boot/vmlinuz-lts
INITRD /boot/intel-ucode.img,/boot/initramfs-lts

LABEL old
KERNEL /boot/vmlinuz-lts-old
INITRD /boot/intel-ucode.img-old,/boot/initramfs-lts-old

LABEL new
KERNEL /boot/vmlinuz-lts-new
INITRD /boot/intel-ucode.img-new,/boot/initramfs-lts-new
EOF
