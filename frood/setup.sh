#!/bin/sh
set -e

rc-update add devfs sysinit
rc-update add dmesg sysinit
rc-update add udev sysinit
rc-update add udev-trigger sysinit
rc-update add udev-settle sysinit

rc-update add udev-postmount default

rc-update add hwclock boot
rc-update add modules boot
rc-update add sysctl boot
rc-update add hostname boot
rc-update add bootmisc boot
rc-update add syslog boot
rc-update add klogd boot
rc-update add networking boot
rc-update add seedrng boot
rc-update add zfs-import boot
rc-update add tler boot

rc-update add mount-ro shutdown
rc-update add killprocs shutdown

ln -s /etc/init.d/agetty /etc/init.d/agetty.ttyS0
ln -s /etc/init.d/agetty /etc/init.d/agetty.tty1

rc-update add agetty.ttyS0 default
rc-update add agetty.tty1 default

rc-update add acpid default
rc-update add crond default
rc-update add local default
rc-update add openntpd default
rc-update add smartd default
rc-update add zfs-zed default
rc-update add sshd default
rc-update add tailscale default
rc-update add srvmonitor default

chpasswd -e <<'EOF'
root:$6$twsDxnP.TG2M8J4l$7lte7E/ImK4UwoursD7qQCC7XMUothIDb9FTH1MncxYbGQDUQPkC/9pxleTwPxEs3nbatApszxuwc4yj6ucdX1
EOF

sed -i 's/^ ?-nobackups/-nobackups/' /etc/joe/joerc
sed -i 's/^ ?-nodeadjoe/-nodeadjoe/' /etc/joe/joerc

# For zpool-import. https://abyssdomain.expert/@filippo/113382158651811946
zgenhostid 000f300d

# Remove the human-unfriendly /dev/disk/by-id WWN symlinks.
# https://github.com/eudev-project/eudev/issues/289
cat /usr/lib/udev/rules.d/60-persistent-storage.rules | \
    grep -v 'SYMLINK+="disk/by-id/nvme-$attr{wwid}' | \
    grep -v 'SYMLINK+="disk/by-id/wwn-' > /etc/udev/rules.d/60-persistent-storage.rules
