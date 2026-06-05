---
title: Certificate Transparency Uptime Alerts
canonical: https://uptime.geomys.org/ct/
---

# Certificate Transparency Uptime Alerts

This is a little service that makes it possible to set up alerting for
[endpoint_uptime_24h.csv](https://www.gstatic.com/ct/compliance/endpoint_uptime_24h.csv).

e.g. `https://uptime.geomys.org/ct/24h/geomys.org` will return a 503 if any
lines matching "geomys.org" have an uptime column below 95 (for `/add-chain` and
`/add-pre-chain`, which are queried by Google every hour) or 99 (for all other
endpoints, which are queried by Google every 10 minutes).

You can use it with any filter string, and it also takes a parameter like
`?threshold=98`. You're welcome to use our instance, but no guarantees!

We recommend setting up alerting such that the endpoint is checked from multiple
locations, and you are only alerted if all locations hit 503s for more than 15 minutes.

There is also a witness monitoring service at
[https://uptime.geomys.org/witness/](https://uptime.geomys.org/witness/).
