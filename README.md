iodine-dockerfile
=================

Self-contained Dockerfile for Iodine, the TCP-over-DNS tunnel.
[http://code.kryo.se/iodine/](http://code.kryo.se/iodine/)

```
sudo docker pull filosottile/iodine
sudo docker run -privileged -p 53:53/udp -e IODINE_HOST=t.example.com -e IODINE_PASSWORD=qwerty filosottile/iodine
```

Note that you will need to setup your DNS for this to work.
