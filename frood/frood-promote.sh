#!/bin/sh
set -ex

if [ ! -e /media/usb/boot/vmlinuz-lts-new ]; then
    echo "No new image to promote"
    exit 1
fi

# Note this could be made safer using the BOOT_IMAGE kernel command line parameter.

mv /media/usb/boot/vmlinuz-lts /media/usb/boot/vmlinuz-lts-old
mv /media/usb/boot/initramfs-lts /media/usb/boot/initramfs-lts-old
mv /media/usb/boot/intel-ucode.img /media/usb/boot/intel-ucode.img-old

mv /media/usb/boot/vmlinuz-lts-new /media/usb/boot/vmlinuz-lts
mv /media/usb/boot/initramfs-lts-new /media/usb/boot/initramfs-lts
mv /media/usb/boot/intel-ucode.img-new /media/usb/boot/intel-ucode.img

extlinux --clear-once /media/usb/boot/syslinux
