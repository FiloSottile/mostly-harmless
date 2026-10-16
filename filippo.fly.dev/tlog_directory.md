---
title: tlog.directory
canonical: https://tlog.directory
---

# tlog.directory

Transparency Logs are a new piece of fundamental digital infrastructure that can be used to create honest and auditable public distribution of information.

A Transparency Log is a scalable, efficient append-only record system,
with algorithms at the core that have been designed such that *other people can easily verify* that the log has been append-only at all times.
Moreover, a set of interoperable specifications raises Transparency Logs into the
realm of technology that's a good choice because it's *well known*
and there is a growing community of tools, implementations, auditors, and users who are familiar with it.

Transparency Logs make it possible to hold individuals and organizations **_accountable_** for their claims.
They're a useful tool for discouraging bad behaviors by making consequences and conversations possible.

[tlog.directory](https://tlog.directory/) -- this page! -- is an index of information on Transparency Logs!
We'll keep it brief, and link you to more information.

#### table of contents

- ⇨ [Introductions and Primers](#introductions-and-primers), for you who's just learning
- ⇨ [Specifications](#specifications)
- ⇨ [Implementations and Libraries](#implementations-and-libraries)
- ⇨ [Case Studies and Known Deployments](#case-studies-and-known-deployments)
- ⇨ [Glossary](#glossary) of common terms
	- ... with an extra long section on [Witnesses vs Monitors](#witnesses-vs-monitors),
		a subtle topic that explains a lot of the technical magic that make Transparency Logs special.
- ⇨ [Frontiers](#frontiers) -- work that's pushing Transparency Logs into the future and into more domains, with ready tools and research solutions.
	- [Collections and Indexes](#collections-and-indexes)
	- [Key Transparency](#key-transparency)
	- [Advice for Log Entry Design](#advice-for-log-entry-design)
- ⇨ [Disambiguations](#disambiguations) (tl;dr tlogs aren't CT logs.  They're related but more general.)
- ⇨ [Community](#community) -- where to find it!



## Introductions and Primers

- [WTF is a Transparency Log?  A Brief Primer.](https://warplog.leaflet.pub/3mkiaduggkc2c)



## Specifications

- [`c2sp.org/tlog-tiles`](https://c2sp.org/tlog-tiles)-- Specification of "tiled" static asset-based transparency logs.
- [RFC6962 § 2.1](https://datatracker.ietf.org/doc/html/rfc6962#section-2.1) -- defines merkle hash trees as used in standard Transparency Logs, detailing the node and leaf formats, the hash function used, and the selection of nodes required for inclusion and tree extension proofs.  (Sections outside of 2.1 concern a (legacy!) form of Certificate Transparency, and are not relevant to Transparency Logs.)
- [`c2sp.org/tlog-checkpoint`](https://c2sp.org/tlog-checkpoint) -- Specification of Checkpoints -- an interoperable format for signed transparency log tree heads.
- [`c2sp.org/tlog-witness`](https://c2sp.org/tlog-witness) -- Specifies a protocol (over HTTP) to obtain transparency log witness cosignatures.
- [`c2sp.org/tlog-proof`](https://c2sp.org/tlog-proof) -- describes how to use Transparency Logs to then produce offline-verifiable transparency log proofs (also known as "_spicy signatures_").

These works are also notable parts of the foundation and early history of Transparency Logs:

- [Transparent Logs for Skeptical Clients (2019)](https://research.swtch.com/tlog) -- a detailed introduction to transparency logs and their uses, and the first introduction of _tile based_ transparency logs.
- [Verifiable Data Structures (2015)](https://github.com/google/trillian/blob/master/docs/papers/VerifiableDataStructures.pdf) -- provides an overview of Verifiable Logs, Verifiable Maps, and the synthesis of Verifiable Log-Backed Maps, with a summary of operations possible on them and their relative resource costs.



## Implementations and Libraries

There are several implementations of Transparency Logs.
They come as both complete product solutions, and as libraries you can use to build your own.

### logs

- Tessera -- https://github.com/transparency-dev/tessera/
	- Tessera is a library for easily creating your own (Tiled) Transparency Logs.
- Sigsum -- https://www.sigsum.org/
	- Sigsum is a log implementation, as well as a log service provider, where log entries are associated with authors defined by public keys.
- SigStore / Rekor -- https://docs.sigstore.dev/logging/overview/
	- SigStore is a log implementation where log entries are associated with authors as defined by OIDC interactions that the log operator is trusted to have verified at the time of submission.
- Golang's SumDB -- https://pkg.go.dev/golang.org/x/mod/sumdb/tlog
	- An honorable mention: this particular Transparency Log is specific to the Golang module ecosystem.

There is considerable diversity and simultaneously interoperability between Transparency Log implementations.
We keep it to one line of description apiece here: consult each project's documentation for more details!

A few factors to keep in mind, though:

- Some projects create *single* logs; others have built-in provisions for multi-tenancy, based on public keys, certificates, or other structures.
- Some Transparency Log systems have *structured* log entries, which have confined specific formats; others enforce no structure at all.

### libraries

- In Golang:
	- [golang.org/x/mod/sumdb/tlog](https://pkg.go.dev/golang.org/x/mod/sumdb/tlog) is reusable code that backs the Golang SumDB Transparency Log.
	- [filippo.io/torchwood](https://pkg.go.dev/filippo.io/torchwood) further extends the `sumdb/tlog` libraries.
- (This is not an exhaustive list!  There are already more libraries than we can easily keep track of; Please help expand this list!)

### tools

Witnessing:

- [torchwood litewitness](https://github.com/FiloSottile/torchwood/blob/main/cmd/litewitness/README.md) -- implements a Witness per the Transparency Log standards.

Observability and Inspection:

- [Woodpecker](https://github.com/mhutchinson/woodpecker/) is a CLI tool for inspecting Transparency Log contents.  It works over most standard [Tiled](https://c2sp.org/tlog-tiles) logs (and is easily extended for new transports if not already supported).
- [Woodpecker Web](https://github.com/transparency-dev/incubator/tree/main/woodpecker-web) is a web-based tool for inspecting Transparency Log contents.  Like the CLI program of similar name, it works over most standard [Tiled](https://c2sp.org/tlog-tiles) logs.  It's very easy to deploy: it's a static html page, and accesses the rest of the log over relative http paths.
	- See https://nyx.n621.de/sigsum/barreleye/ for a live instance of this!

### demos

- [tessera posix oneshot](https://github.com/transparency-dev/tessera/tree/main/cmd/examples/posix-oneshot) -- a minimal educational example of using Tessera as a library to produce your own independent Transparency Log, entirely on the local machine, with no substantial system dependencies.
- [spicytool](https://github.com/warpfork/spicytool/) -- Demo implementation of `tlog-proof`.
	- (this is a very small codebase and may be another good introductory walk through code working with tlogs and tessera!)



## Case Studies and Known Deployments

- [Securing the Public Go Module Ecosystem (2019)](https://golang.org/design/25530-sumdb) is a large, real-world and fully deployed example of Transparency Log rollout and application.
	- This document is the original design plan for the now-deployed Golang SumDB system, which elucidates the design goals and works through how the transparency log system provides value to the Go package distribution ecosystem.
	- The system described here is now in production and involved in the serving of effectively all Go package distribution.

- [GopherWatch](https://www.gopherwatch.org/) ([source](https://github.com/mjl-/gopherwatch)) -- an example of a Monitor: this service tracks the Go SumDB log, monitors it, and creates notifications based on changes.

- Modern "static" Certificate Transparency now uses Transparency Logs!
	- https://sunlight.dev/ ([source](https://github.com/FiloSottile/sunlight)) is a full implementation of such a static CT log.  See also the documentation from its introduction: https://filippo.io/a-different-CT-log covers both this new TLog-based system, and the differences from previous systems.



## Glossary

- **Checkpoint** -- a document describing a tip of a Transparency Log.
	Includes the size of the log, and the hash of the current tip.
	Also typically bundles signatures from several Witnesses.
	See [c2sp.org/tlog-checkpoint](https://c2sp.org/tlog-checkpoint) for specification.
- **Monitor** -- a role in a transparency log ecosystem that observes a transparency log and the data within it.
	Monitors generally have some understanding of the data in the log (in contrast to Witness roles)
	and may provide alerting based on contents.
	Also in contrast to Witnesses, Monitors generally receive the full contents of the log.
	See the [Witnesses vs Monitors](#witnesses-vs-monitors) section for more details.
- **Observer** -- colloquially, anyone who looks at any data in a log.
	It's a superset of Monitors, Witnesses, and... perhaps you!
	Technical documentation will usually use more specific terms.
- **Split View** -- describes an undesirable situation in which some malicious system prevents different data to different parties depending on who is asking.
	(Transparency Logs, combined with Witnessing, are designed to combat this --
	either preventatively (when data is unaccepted until it's certified to be seen by a quorum of witnesses),
	or consequentially (because any split view of a transparency log presented to any observer is also a self-contained proof of malfeasance).)
- **Tiles** -- a pattern of files that specifies storing tlog data in predictable ways per file and in sharded directories,
	which is easily deterministically accessed and efficient to sync.
	Tiling was first described by [Russ Cox (2019)](https://research.swtch.com/tlog#tiling_a_log);
	see [`c2sp.org/tlog-tiles`](https://c2sp.org/tlog-tiles) for an up-to-date specification.
- **Witness** -- a role in a transparency log ecosystem that observes a log and signs statements that the log has been append-only at all points the witness observes it.
	Witnessing is a very low-cost operation, and can operate even without seeing the full log (contrast: Monitoring).
	See the [Witnesses vs Monitors](#witnesses-vs-monitors) section for more details.
- **Verifiable Log** -- synonym of Transparency Log.
	Seen in some older documents; we now mostly say "Transparency Log".


### Witnesses vs Monitors

Witnessing and Monitoring are two distinct roles in a Transparency Log's ecosystem.
They sound very similar, because they're both observing the log!
The distinction is in how much work they do:
Witnesses perform a single, very _specific_ task -- it's something that's clear, very standardized, very optimized, and _common to all logs_.
Monitoring is a term to describe a broader range of operations that can be done while observing a log (not all of which are standardized, and not all of which can be optimized in the same way as Witnessing is).

**Witnessing** means very specifically: remembering a log's previous size and tree head hash; and when being informed of a new size and tree head for that log, and given a merkle extension proof that shows the previous tree head hash being included in the merkle tree of the new one... signs a message to certify the witness observed that.

Why so specific?

- Witnesses need `O(1)` state per log they witness -- that means they're very cheap to run, and one Witness can easily be the witness for many (many) logs, while having very little need for persistent data (and, the need does not grow over time!).
- Witnesses work by compact transfer of merkle proofs -- practically speaking, that means kilobytes (or less) of data transfer needed to perform a witnessing operation.  This remains true regardless of how many log updates are being witnessed in one operation (or more specifically, it requires `log(N)` data -- proportional to the number of log events... and importantly, is *not* affected by the size of actual log entries.)
- By being so specific, the Witnessing protocol is a common standard.
	- Witnesses can operate without knowing any specific details about a transparency log's contents -- this makes them a collectively shareable resource!
	- Witnessing has a sufficiently consistent semantic meaning that the Transparency Log itself can re-aggregate witnessing data and re-publish it meaningfully.
- Witnessing happens in **real time** (and so *has to be* dirt cheap and very fast)... because it is the blocking gate for the log moving forward!  (Ecosystemically, tools generally default to not considering a log's records to be authoritative until they've been confirmed by some quorum of Witnesses.  This is how a Transparency Log combined with its Witnesses becomes resistant to split-view attacks.)

**Monitoring** is a more general description: it means _any_ process that observes the Transparency log, and also it typically refers to processes that actually look at the full log entries (not just the log's merkle tree).

What does this generality imply?

- For starters, Monitors typically take significantly more resources to run: transferring all the data entries that a log references can describe a lot of data, and it grows linearly as the log grows -- that's a big contrast to the small and only logarithmically growing data transfer costs of Witnessing.
- ... but in exchange, they can do a lot more:
	- Monitors can provide alerting and notification services to users based on the full contents of logs.
	- Monitors over logs that use features like Verifiable Indexes and other operation-logged algorithms can verify the lawful application of the algorithms, and notify on transgressions.
	- Monitors can be involved in the full mirroring of logs, in cases of low trust where adversarial replication is desirable.
- Signatures from Monitors _are not_ re-aggregated by a Transparency Log in a standardized way -- this is in significant contrast to Witnessing claims, which are.  The reason for this is that because monitoring is a broader and less specifically formalized concept, there's not enough semantic clarity about what such aggregation and republication would mean, nor clear automatable reactions that could be based on it.

Monitoring and Witnessing are both important stories in a Transparency Log's journey to relevance.
Neither can fully cover the purposes of the other.



## Frontiers

### Collections and Indexes

There are several ways to store collections (usually, associative maps)
that connect with and benefit from Transparency Logs.

- Merkle Search Trees (and variants) -- where the MST is map structure, and is composed with a TLog simply by being referred to by the map's root hash being published as an entry in a TLog leaf.
- Verifiable Indexes -- similar in concept to an MST, but instead of being referred to within the TLog leaves, the relationship is flipped: a Vindex is freestanding, and at its leaves, an index into a TLog is stored.
	- Sometimes also-known-as "Transparent Maps".
	- This results in slightly more indirect access patterns to read, but is simple to audit and monitor, and architecturally is convenient to add to already-running tlogs.
	- See [transparency-dev/incubator/vindex](https://github.com/transparency-dev/incubator/tree/main/vindex) for one implementation of this pattern.
		- See [docs subdir of vindex](https://github.com/transparency-dev/incubator/tree/main/vindex/docs/v1) for design notes towards future iterations.

### Key Transparency

[Key Transparency](https://en.wikipedia.org/wiki/Key_Transparency) is the umbrella term
for systems that make identity and keys associated with identities into a transparently logged system.

Transparency Logs are an important _component_ of Key Transparency system design,
but there are many further subtleties involved in Key Transparency.

Search out further literature on this topic if you're working in or near it!


### Advice for Log Entry Design

When implementing use of Transparency Logs, and deciding exactly what content to log, keep in mind that Transparency Logs are (by design) very difficult to redact.
This cuts in two ways: it's both the fundamental purpose of a TLog,
but it also means publishing to the log should be careful not to introduce any content that might need future redaction.

Typically, the solution pattern to this challenge is: publish a hash of the message that the Transparency Log is providing nonrepudiability to, rather than publishing that message itself directly in the log's leaf, and publish the message separately.
Then, redaction is as simple as dropping that message from other places its stored;
at the same time, anyone monitoring the log may have kept their own copy,
so if this redaction is malfeasance, it can still be called out and seen as such.

When considering if some part of the logged data should go in a separate, hash-referenced message vs in the log leaf entry directly, also remember that anything you might want to partition or index the log by must still be in the entry leaf directly.
(If some piece of data is in the referenced message, and redaction is attempted for that message, then any kinds of indexing built over the log would become unverifiable, which substantially reduces the utility of that log.)

(There is not yet a standard for this pattern, but tentative discussions exist on standardizing an "adjunct message" format, which would make implementing this pattern easier, and would allow for mirroring tools to automatically replicate such off-log messages.)



## Disambiguations

Transparency Logs are sometimes confused with Certificate Transparency (CT) efforts.
This is understandable, because they're closely related -- much of the latest work in CT is based on Transparency Logs! -- and many of the same people are involved.
However, they are two separate topics.

Transparency Logs are much more general than Certificate Transparency.
CT involves many specifications which are in some cases noticeably older than those of Transparency Logs,
and are not necessarily how people would've done things if it was done again today.
CT is undergoing a modernization called "MCT" (short for Merkle-tree Certificate Transparency)
which is generally descriptive of rebuilding the concepts of CT _on top of_ Transparency Logs.
At the same time, parts of Transparency Log specifications do also still refer to some of the RFCs about CT for algorithmic details.

This section is not to throw shade at Certificate Transparency!
However, confusion does sometimes arise about the relationship between the two.
The important takeaways are:

- understanding Transparency Logs does *not* require understanding Certificate Transparency,
- and Transparency Logs are good for *much more* than certificates.

If you *are* interested in Certificate Transparency, and the work in that area which combines it with Transparency Logs:

- [c2sp.org/static-ct-api](https://c2sp.org/static-ct-api) is the specification for modern "static CT" using Transparency Logs.
- [sunlight.dev](https://sunlight.dev/) is one of the live implementations of "static CT".
	- [filippo.io/a-different-CT-log (2023)](https://filippo.io/a-different-CT-log) introduces what became the Sunlight project, and is a great coverage of motivations.



## Community

Transparency Logs are still being actively developed, and also have an enthusiastic community who will help push the boundaries when needed, and help with understanding and rollout wherever possible.

Here are a few of the places you can find us:

- [Transparency-Dev Slack](https://transparency.dev/slack/)
- The Transparency Dev group has approximately-monthly meetings:
	- [meeting notes (google doc)](https://docs.google.com/document/d/1cQop8_p7-fV5CEO5ADyvLrDGDm8BR79MytMqc0MeAeY/edit?usp=sharing)
	- [events calendar (google calendar)](https://calendar.google.com/calendar/u/0?cid=Y181Nzg0NDRkZDc5Y2NiMmNhNmY0ZmU0Yjc3ZjA3ZmZmOWY3NzVlMTg2NmM3MjEwOWZlYzE2ZDJkNTJhNGU1ZmVkQGdyb3VwLmNhbGVuZGFyLmdvb2dsZS5jb20)
	- [specific event (google calendar)](https://calendar.google.com/calendar/event?action=TEMPLATE&tmeid=NDlyc25rY3JwaWhtcmxhMWdxaTI2Z2lpNzBfMjAyNjA0MjdUMTUwMDAwWiBjXzU3ODQ0NGRkNzljY2IyY2E2ZjRmZTRiNzdmMDdmZmY5Zjc3NWUxODY2YzcyMTA5ZmVjMTZkMmQ1MmE0ZTVmZWRAZw&tmsrc=c_578444dd79ccb2ca6f4fe4b77f07fff9f775e1866c72109fec16d2d52a4e5fed%40group.calendar.google.com)
	- [meeting link (google meet)](https://meet.google.com/tjq-akzg-qwc)
- [witness-network.org](https://witness-network.org) -- this is a hub for organizing public infrastructure for Witnessing.

Other smaller / differently-focused working groups do exist!
You can probably find them starting from the above largest public one.

If you host a gathering or chat venue you'd like listed here, please get in touch :)
