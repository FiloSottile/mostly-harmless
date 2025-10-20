FROM golang:1.25.0-alpine3.22 AS builder

COPY . src
WORKDIR src
ENV CGO_ENABLED=0
RUN go install -trimpath

FROM alpine:3.22

COPY --from=builder /go/bin/ct-uptime-alerts /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/ct-uptime-alerts"]
