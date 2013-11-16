FROM ubuntu
MAINTAINER Patrick O'Doherty <p@trickod.com>

EXPOSE 9091
ENV VERSION 0.2.3.25

RUN apt-get install -y curl build-essential libevent-dev libssl-dev
RUN curl https://www.torproject.org/dist/tor-${VERSION}.tar.gz | tar xz -C /tmp

RUN cd /tmp/tor-${VERSION} && ./configure
RUN cd /tmp/tor-${VERSION} && make
RUN cd /tmp/tor-${VERSION} && make install

ADD ./torrc /etc/torrc

# Generate a random nickname for the relay
RUN echo "Nickname docker$(head -c 16 /dev/urandom  | sha1sum | cut -c1-10)" >> /etc/torrc

CMD /usr/local/bin/tor -f /etc/torrc
