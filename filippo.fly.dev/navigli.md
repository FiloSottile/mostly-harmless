---
title: The Navigli Witness
canonical: https://geomys.org/witness/navigli
---

# The Navigli Staging Witness

`witness.navigli.sunlight.geomys.org` is a staging transparency log witness
operated by [Geomys](https://geomys.org/). It implements the
[c2sp.org/tlog-witness](https://c2sp.org/tlog-witness) API.

For the verifier key and configured [witness-network.org log
lists](https://witness-network.org/log-lists/), see [the system's
homepage](https://navigli.sunlight.geomys.org/). It generally configures all
Staging and Testing log lists. If you wish to have your log witnessed, follow
[the process to add it to the witness-network.org log
lists](https://witness-network.org/participate/). The lists are updated
automatically every 15 minutes.

This is a **staging instance**, and in particular it's the staging instance of
the Sunlight developers, so it has decent odds of encountering bugs. There is
uptime monitoring, but it is not connected to phone pagers. Aside from that, we
plan to keep operating it for the foreseeable future.

It runs the latest development version of [Sunlight](https://sunlight.dev/), in
the same process as the staging Geomys *Navigli* Certificate Transparency logs.
It is hosted on the same machine as the production Geomys *Tuscolo* logs, which
is described in the [log's announcement][]. The machine's config files are
[publicly available](https://config.sunlight.geomys.org/). Keys are stored on an
encrypted filesystem, which is unlocked manually at boot by Geomys staff, but
are otherwise simple software keys.

[log's announcement]: https://groups.google.com/a/chromium.org/g/ct-policy/c/KCzYEIIZSxg/m/zD26fYw4AgAJ

Prometheus metrics are publicly available at
[navigli.sunlight.geomys.org/metrics](https://navigli.sunlight.geomys.org/metrics)
(look for the `sunlight_witness_` prefix). They include per-log request
counters, errors, and abserved log sizes, which might be useful for debugging.

<!-- for prod, mention https://status.sunlight.geomys.org/ -->
