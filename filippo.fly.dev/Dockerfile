FROM golang:1.18-alpine AS builder

COPY . src
WORKDIR src
ENV CGO_ENABLED 0
RUN go install -trimpath

FROM alpine:3.15

COPY --from=builder /go/bin/filippo.fly.dev /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/filippo.fly.dev"]
