# Use phusion/baseimage as base image.
FROM phusion/baseimage:0.9.5

MAINTAINER Filippo Valsorda <fv@filippo.io>

# Set environment variables.
ENV HOME /root

# Use baseimage-docker's init system.
CMD ["/sbin/my_init"]

RUN apt-get install -y curl build-essential libevent-dev libssl-dev
RUN curl https://www.torproject.org/dist/tor-${VERSION}.tar.gz | tar xz -C /tmp

RUN cd /tmp/tor-${VERSION} && ./configure
RUN cd /tmp/tor-${VERSION} && make
RUN cd /tmp/tor-${VERSION} && make install

ADD ./torrc /etc/torrc.template

# Allow you to upgrade without losing the service details
VOLUME /var/tor

# Add the boot script (this will link to the server)
RUN mkdir -p /etc/my_init.d
ADD boot.sh /etc/my_init.d/tor-boot.sh

# Add the tor service entry for runit
RUN mkdir /etc/service/tor
ADD tor.sh /etc/service/tor/run

# Clean up APT when done.
RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
