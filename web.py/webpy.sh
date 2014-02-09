#!/bin/sh

exec python /root/serve.py 80 >> /root/serve.log 2>&1
