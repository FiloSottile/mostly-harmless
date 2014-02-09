# Use phusion/baseimage as base image.
FROM phusion/baseimage:0.9.5

MAINTAINER Filippo Valsorda <fv@filippo.io>

# Set environment variables.
ENV HOME /root
RUN /etc/my_init.d/00_regen_ssh_host_keys.sh

# Install the SSH key
ADD authorized_keys /tmp/authorized_keys
RUN cat /tmp/authorized_keys >> /root/.ssh/authorized_keys && rm -f /tmp/authorized_keys

# Use baseimage-docker's init system.
CMD ["/sbin/my_init"]

# Install dependencies
RUN apt-get install -y python-pip
RUN pip install web.py

ADD serve.py /root/serve.py

# Expose web port
EXPOSE 80

# Add the service entry for runit
RUN mkdir /etc/service/webpy
ADD webpy.sh /etc/service/webpy/run

# Clean up APT when done.
RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
