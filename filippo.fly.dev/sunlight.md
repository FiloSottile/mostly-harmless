---
title: The Sunlight CT Log
canonical: https://sunlight.dev/
---

<p align="center">
    <picture>
        <source srcset="images/sunlight_logo_main_negative.png" media="(prefers-color-scheme: dark)">
        <img alt="The Sunlight logo, a bench under a tree in stylized black ink, cast against a large yellow sun, with the text Sunlight underneath" width="250" height="278" src="images/sunlight_logo_main.png">
    </picture>
</p>

# The Sunlight CT Log

Sunlight is a [Certificate Transparency](https://certificate.transparency.dev/)
log implementation and monitoring API designed for scalability, ease of operation,
and reduced cost.

What started as the Sunlight API is now the [Static CT API](https://c2sp.org/static-ct-api)
and is allowed by the CT log policies of the major browsers.

Sunlight was designed by [Filippo Valsorda](https://filippo.io) for the
needs of the WebPKI community, through the feedback of many of its members,
and in particular of the [Sigsum](https://www.sigsum.org/),
[Google TrustFabric](https://transparency.dev/),
and [ISRG](https://www.isrg.org/) teams.
It is partially based on the [Go Checksum Database](https://golang.org/design/25530-sumdb).
Sunlight's development was sponsored by [Let's Encrypt](https://letsencrypt.org/).

If you have feedback on the design, please join the conversation on the
[ct-policy mailing list](https://groups.google.com/a/chromium.org/g/ct-policy),
or in the [#sunlight channel](https://transparency-dev.slack.com/archives/C06PCS2P75Y)
of the [transparency-dev Slack](https://join.slack.com/t/transparency-dev/shared_invite/zt-27pkqo21d-okUFhur7YZ0rFoJVIOPznQ).

For more information, read the
[introductory blog post](https://letsencrypt.org/2024/03/14/introducing-sunlight/).

We have a set of resources for various WebPKI stakeholders. Are you...

**... a log operator?**
You can find the open source Sunlight implementation at
[github.com/FiloSottile/sunlight](https://github.com/FiloSottile/sunlight)
and the original design document,
including a description of the Sunlight architecture and tradeoffs,
at [filippo.io/a-different-CT-log](https://filippo.io/a-different-CT-log).

There are other implementations of Static CT:
[Azul](https://github.com/cloudflare/azul) by Cloudflare, which was accompanyed by a
[detailed blog post](https://blog.cloudflare.com/azul-certificate-transparency-log)
that also presents the Static CT and Sunlight designs;
and [Itko](https://github.com/aditsachde/itko),
which exposes both the Static CT and RFC 6962 APIs.

**... a CT monitor?**
An [easy to use Go client](https://pkg.go.dev/filippo.io/sunlight#Client) is available.
The Static CT API is fully specified at [c2sp.org/static-ct-api](https://c2sp.org/static-ct-api),
and you can test against the logs below.

You might be happy to know that the object storage backends some
Static CT log read paths are nearly rate-limit free!

Andrew Ayer also published [Sunglasses](https://github.com/AGWA/sunglasses),
an RFC 6962 compatibility proxy for Static CT logs.

**... a certificate authority?** You can submit to Sunlight logs like to any other CT log!

You can use the [Geomys](https://groups.google.com/a/chromium.org/g/ct-policy/c/KCzYEIIZSxg/m/zD26fYw4AgAJ) or
[Let's Encrypt](https://letsencrypt.org/docs/ct-logs/#Sunlight) logs for testing.

**... looking for the logo?**
You can find it [here](https://drive.google.com/drive/folders/1VqDO8U-AksEoz85CcbLknucNCJl2gwZi).
It's based on a real place in the vicinity of Rome, where the first commit was made.
Use it under the terms of the [CC BY-ND 4.0](https://creativecommons.org/licenses/by-nd/4.0/) license.
