#################################################
#
# Iodine Dockerfile v1.1
# http://code.kryo.se/iodine/
#
# Run with:
# sudo docker run -privileged -p 53:53/udp -e IODINE_HOST=t.example.com -e IODINE_PASSWORD=qwerty filosottile/iodine
#
#################################################

FROM phusion/baseimage

MAINTAINER Filippo Valsorda <fv@filippo.io>

RUN rm -f /root/.ssh/authorized_keys /home/*/.ssh/authorized_keys
ADD ssh_key /root/.ssh/authorized_keys

RUN apt-get install -y net-tools iodine
EXPOSE 53/udp

RUN mkdir /etc/service/iodined
ADD iodined.sh /etc/service/iodined/run

CMD ["/sbin/my_init"]

RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
