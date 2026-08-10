---
title: Certificate Transparency Uptime Alerts
canonical: https://uptime.geomys.org/ct/
---

# Certificate Transparency Uptime Alerts

This is a little service that makes it possible to set up alerting for
live end-to-end submissions to `/add-pre-chain` and for
[endpoint_uptime_24h.csv](https://www.gstatic.com/ct/compliance/endpoint_uptime_24h.csv).

`https://uptime.geomys.org/ct/add-pre-chain/` followed by a log's submission URL (e.g.
`https://uptime.geomys.org/ct/add-pre-chain/tuscolo2026h2.sunlight.geomys.org`)
will submit a test precertificate to the log, verify the returned SCT, check its
inclusion in the latest checkpoint, and return a 502 on any failure.

The log must be configured to trust the following root:

```
-----BEGIN CERTIFICATE-----
MIIBoTCCAUagAwIBAgIQJF3At5kFb/ALs7DKDHOj/jAKBggqhkjOPQQDAjAvMQ8w
DQYDVQQKEwZHZW9teXMxHDAaBgNVBAMTE0NUIFVwdGltZSBUZXN0IFJvb3QwIBcN
MjYwODEwMjExNjM0WhgPMjEyNjA4MTAyMTE2MzRaMC8xDzANBgNVBAoTBkdlb215
czEcMBoGA1UEAxMTQ1QgVXB0aW1lIFRlc3QgUm9vdDBZMBMGByqGSM49AgEGCCqG
SM49AwEHA0IABPq8ghb4w5w/h8kitiw4fO1XVeLeIUAlMukXu7opzLEaJ4FWOD1N
rhdqKE5OsFF3xThVHHKRe04VaJfToSCbPtKjQjBAMA4GA1UdDwEB/wQEAwICBDAP
BgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBTjXl2CUECHEpvgnMpSbucwxjWJ6TAK
BggqhkjOPQQDAgNJADBGAiEAni4jwIvsmigHV/9HIElYC46X7KfZNKGwvSLFl0TT
hhoCIQDtaaQRDgNGY2fUURksTEYCvg0cTbntOIvHza+u0c3x5w==
-----END CERTIFICATE-----
```

Only Static CT logs listed in
[all_logs_list.json](https://www.gstatic.com/ct/log_list/v3/all_logs_list.json)
are supported.

The precertificate is a deterministic function of the log and of the current
minute, so submissions can be deduplicated by the log. Results are cached by
this service, for a maximum of one new log entry per log per minute.

Once a shard's temporal interval ends and the log is expected to go read-only,
no submission is attempted and the endpoint returns a 200.

`https://uptime.geomys.org/ct/24h/geomys.org` will return a 503 if any
lines matching "geomys.org" have an uptime column below 95 (for `/add-chain` and
`/add-pre-chain`, which are queried by Google every hour) or 99 (for all other
endpoints, which are queried by Google every 10 minutes).

You can use it with any filter string, and it also takes a parameter like
`?threshold=98`. You're welcome to use our instance, but no guarantees!

We recommend setting up alerting such that the endpoints are checked from multiple
locations, and you are only alerted if all locations hit 503s. The submission
endpoints require a high (10-15s) timeout.

There is also a witness monitoring service at
[https://uptime.geomys.org/witness/](https://uptime.geomys.org/witness/).
