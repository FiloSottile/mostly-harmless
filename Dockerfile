#################################################
#
# Iodine Dockerfile v1.2
# http://code.kryo.se/iodine/
#
# Run with:
# sudo docker run -privileged -p 53:53/udp -e IODINE_HOST=t.example.com -e IODINE_PASSWORD=qwerty filosottile/iodine
#
#################################################

# Use phusion/baseimage as base image.
FROM phusion/baseimage:0.9.5

MAINTAINER Filippo Valsorda <fv@filippo.io>

# Set environment variables.
ENV HOME /root
RUN /etc/my_init.d/00_regen_ssh_host_keys.sh

# Install the SSH key
ADD ssh_key.pub /tmp/ssh_key.pub
RUN cat /tmp/ssh_key.pub >> /root/.ssh/authorized_keys && rm -f /tmp/ssh_key.pub

# Install iodine
RUN apt-get install -y net-tools iodine

# Add the runit iodine service
RUN mkdir /etc/service/iodined
ADD iodined.sh /etc/service/iodined/run

# Bind to the host 53 UDP port
EXPOSE 53:53/udp

# Use baseimage-docker's init system.
CMD ["/sbin/my_init"]

# Clean up APT when done.
RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
