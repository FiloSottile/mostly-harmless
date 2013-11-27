#################################################
#
# Iodine Dockerfile v1.0
# http://code.kryo.se/iodine/
#
# Run with:
# sudo docker run -privileged -p 53:53/udp -e IODINE_HOST=t.example.com -e IODINE_PASSWORD=qwerty filosottile/iodine
#
#################################################

FROM ubuntu

MAINTAINER Filippo Valsorda <fv@filippo.io>

# Update APT cache
RUN echo "deb http://archive.ubuntu.com/ubuntu precise main universe" > /etc/apt/sources.list
RUN apt-get update  # 2013-11-26

# Set the locale
RUN apt-get install -y language-pack-en
RUN update-locale LANG=en_US.UTF-8

RUN apt-get install -y net-tools iodine

EXPOSE 53/udp

# Thanks to https://github.com/jpetazzo/dockvpn for the tun/tap fix
CMD ["/bin/bash", "-c", "mkdir -p /dev/net && mknod /dev/net/tun c 10 200 && iodined -f 10.16.0.1 $IODINE_HOST -P $IODINE_PASSWORD"]
