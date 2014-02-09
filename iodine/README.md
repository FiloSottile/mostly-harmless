iodine-dockerfile
=================

Self-contained Dockerfile for Iodine, the TCP-over-DNS tunnel.
[http://code.kryo.se/iodine/](http://code.kryo.se/iodine/)

```
git clone https://github.com/FiloSottile/Dockerfiles.git
cp ~/.ssh/authorized_keys Dockerfiles/iodine/
sudo docker build -t="filosottile/iodine" Dockerfiles/iodine
sudo docker run -d -privileged -p XX.XX.XX.XX:53:53/udp \
 -e IODINE_HOST=t.example.com -e IODINE_PASSWORD=qwerty filosottile/iodine
```

Then on the client you can run `iodine` and then `ssh root@10.16.0.1`

Note that you still need to setup your DNS for this to work.
