# docker-web.py

This is a sample Dockerfile showing how to use [`phusion/baseimage`](https://github.com/phusion/baseimage) to run your own service and a quick-and-easy way to run a container exposing port 80 for testing.

Also, you can use this as a base to offer a really simple web service. (I used it with my Tor .onion image as a client).

**Note:** put you public SSH key in the folder before building, in a file named `ssh_key.pub`. Usually you'll want to copy your `authorized_keys`
