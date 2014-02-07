#!/bin/sh

cp /etc/torrc.template /etc/torrc

# Link to the server, remember to -link servername:server and expose 80
echo "HiddenServicePort 80 $SERVER_PORT_80_TCP_ADDR:80" >> /etc/torrc

cat /etc/torrc
