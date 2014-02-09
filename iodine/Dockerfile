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
ADD authorized_keys /tmp/authorized_keys
RUN cat /tmp/authorized_keys >> /root/.ssh/authorized_keys && rm -f /tmp/authorized_keys

# Install iodine
RUN apt-get install -y net-tools iodine

# Add the runit iodine service
RUN mkdir /etc/service/iodined
ADD iodined.sh /etc/service/iodined/run

# Expose the DNS port, remember to run -p 53:53/udp
EXPOSE 53/udp

# Use baseimage-docker's init system.
CMD ["/sbin/my_init"]

# Clean up APT when done.
RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
