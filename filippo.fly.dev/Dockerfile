FROM golang:1.27.3-alpine3.23 AS builder

COPY . src
WORKDIR src
ENV CGO_ENABLED=0
ENV GOTOOLCHAIN=auto
RUN go install -trimpath

FROM ghcr.io/astral-sh/uv:0.13.5-python3.13-alpine3.23

ENV PATH=/root/.local/bin:$PATH
RUN uv tool install markdown==3.8.2 --with mdx_truly_sane_lists==1.3.0 \
    --with pymdown-extensions==10.16 --with pyyaml==6.0.2 \
    --compile-bytecode --python 3.13

COPY --from=builder /go/bin/filippo.fly.dev /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/filippo.fly.dev"]
