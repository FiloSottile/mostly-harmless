---
title: Geomys FIPS 140-3 Services
canonical: https://geomys.org/fips140
domain: geomys.org
---

<style>
    .inverted {
        /* https://web.dev/articles/light-dark */
        color: Canvas;
        background-color: CanvasText;
    }
    @media (prefers-color-scheme: dark) {
        img {
            filter: invert(1);
        }
    }
    @media print {
        stripe-buy-button {
            display: none !important;
        }
    }
</style>

<p style="text-align: center"><img alt="The Geomys logo, an ink outline of a quaint Italian town on the side of a mountain." width="400" height="160" src="images/geomys_orizz_B_positivo.png">

Geomys handles the [CMVP validation](https://csrc.nist.gov/projects/cryptographic-algorithm-validation-program/details?product=19371) of the **FIPS 140-3 Go Cryptographic Module**, and contributes the module to the upstream Go project, for the benefit of the Go community.

The [FIPS 140-3 Go Cryptographic Module](https://go.dev/doc/security/fips140) is distributed as part of the upstream Go toolchain, where it is seamlessly integrated with the cryptography packages of the Go standard library, and it is replacing the legacy Go+BoringCrypto mode. All FIPS 140-3 approved algorithms in the standard library are implemented by the module, including *post-quantum key exchange algorithm ML-KEM*, and the module is tested on a wide range of platforms.

Geomys ensures the ongoing support and maintenance of the FIPS 140-3 Go Cryptographic Module, including **at least one validation per year** of the latest updates to the standard library cryptographic packages, to comply with new requirements and integrate new features and performance improvements, and **validations of security fixes as needed** for supported versions of the module in response to any vulnerabilities or CVEs that can’t be effectively mitigated with a regular Go version update.

Geomys provides additional commercial services in support of the certification and compliance requirements of Go enterprises, including multiple Fortune 100 companies.

<p style="text-align: center"><img alt="The FIPS 140-3 logo." width="200" height="112" style="padding: 1em 0;" src="images/FIPS 140-3 Logo- BW.png">

## Enterprise Package {#enterprise}

As part of this retainer package, we offer a suite of services that provide a comprehensive and non-disruptive solution to the FIPS 140-3 requirements of Go deployments.

* Inclusion of the customer’s platforms as **Vendor Affirmed Operating Environments** in the Security Policy of *all* upstream FIPS 140-3 Go Cryptographic Module validations submitted during the support period.

* **Rebranded certificates** in the customer’s name for *all* certificates issued for the upstream FIPS 140-3 Go Cryptographic Module during the support period, or authorization letters for the customer to obtain their own rebranded certificate.

* Customizable **attestations** on Geomys letterhead covering various details of in-progress and completed validations.

* **Email support** for queries regarding active and in progress validations, and properties of the FIPS 140-3 Go Cryptographic Module.

* **Preview access** to Security Policies, tested algorithms, Operating Environments, and CMVP lab partner selection, with the opportunity to provide input.

* **Exclusive periodic updates** including progress reports, new version notifications, FIPS 140-3 relevant release notes, news, and upcoming changes.

<p><small>* Inclusion of additional <em>tested</em> Operating Environments and attestations explicitly covering the customer’s products rather than the FIPS 140-3 Go Cryptographic Module may be extra depending on the circumstances.</small></p>

We offer these services exclusively as a time-based retainer to better match the ongoing nature of compliance requirements, and to ensure customers aren’t left behind on unsupported or vulnerable versions of the FIPS 140-3 Go Cryptographic Module.

<p style="text-align: center"><span class="inverted">$80,000/year</span></p>

We work with your procurement process. Email us at fips140@geomys.org.

## Essential Package {#essential}

This lightweight and cost-effective offering provides all the information necessary to effectively integrate the upstream FIPS 140-3 Go Cryptographic Module in your development cycle, with supporting material to answer the questions of customers and auditors.

- **Preview access** to Security Policies, tested algorithms, Operating Environments, and CMVP lab partner selection, with the opportunity to provide input.

- **Exclusive periodic updates** including progress reports, new version notifications, FIPS 140-3 relevant release notes, news, and upcoming changes.

- **Ready-to-download** **attestation** with all the details of in-progress and completed validations, including name and version, Operating Environments, installation instructions, tested algorithms, and CMVP status.

- **Email support** for queries regarding active and in progress validations, and properties of the FIPS 140-3 Go Cryptographic Module. (Up to one query per month.)

<p style="text-align: center">
<span class="inverted">$800/month</span> or <del>$9,600</del> <span class="inverted">$8,640/year</span><br>
(self-service credit card, direct debit, or bank transfer)
</p>

<script async src="https://js.stripe.com/v3/buy-button.js"></script>
<p style="text-align: center">
<stripe-buy-button
  buy-button-id="buy_btn_1ROou5Le4yKccjIdmhFT7OCp"
  publishable-key="pk_live_51Qzj5KLe4yKccjIdbTa2o3AV4DzteqaQYBNMP39GDuPuVM1wJctkQiU6Pb8i1nm5bE3wBIHsdIMjRd8Nm0dksTpe00GzPbJPkD">
</stripe-buy-button>
</p>

<p style="text-align: center">
$10,000/year<br>
(quote / PO / invoice via fips140@geomys.org)
</p>
