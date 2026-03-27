---
title: Witness Uptime Monitoring
canonical: https://uptime.geomys.org/witness/
---

# Witness Uptime Monitoring

This is a little service that submits the following checkpoint to a witness

    geomys.org/witness/test-log
    1
    BCml5C32yqMcl0gjTrcSOeNVx59oPnSdytBzDGBO5k0=

    — geomys.org/witness/test-log wHh/9BPsoBNr2x0Ol3qPBYasIN0HI2ZiBg5ac0v3LQq/7F+YO7U4oWbDeJn1VaWVrlbSEM30Gr7WWYQjj2SBxRoJ/Ao=

and checks that the witness responds with a fresh, valid signature.

The witness can be configured with the following log list

    https://uptime.geomys.org/witness/log-list

or directly with this log vkey

    geomys.org/witness/test-log+c0787ff4+AeMb5VOzy60PTGdGmLPxOKGAa0jNyDGsgv2rnprGju1t

so it will accept the checkpoint.

To run a check, request /witness/add-checkpoint/ followed by the witness vkey, e.g.

    https://uptime.geomys.org/witness/add-checkpoint/witness.navigli.sunlight.geomys.org+a3e00fe2+BNy/co4C1Hn1p+INwJrfUlgz7W55dSZReusH/GhUhJ/G

The vkey name must be the submission prefix of the witness.

We recommend setting up alerting such that the endpoint is checked from multiple
locations, and you are only alerted if all locations hit 503s for more than 15 minutes.

There is also a CT uptime monitoring service at
[https://uptime.geomys.org/ct/](https://uptime.geomys.org/ct/).
