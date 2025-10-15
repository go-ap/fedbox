# FedBOX

[![MIT Licensed](https://img.shields.io/github/license/go-ap/fedbox.svg)](https://raw.githubusercontent.com/go-ap/fedbox/master/LICENSE)
[![Build Status](https://builds.sr.ht/~mariusor/fedbox.svg)](https://builds.sr.ht/~mariusor/fedbox)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-ap/fedbox)](https://goreportcard.com/report/github.com/go-ap/fedbox)

FedBOX is a simple ActivityPub enabled server. Its goal is to serve as a reference implementation for the rest of the [GoActivityPub](https://github.com/go-ap) packages.

It provides the base for some of the common functionality that such a service would require, such as: HTTP handlers and middlewares, storage and filtering etc.

The current iteration can persist data to [BoltDB](https://go.etcd.io/bbolt), [Badger](https://github.com/dgraph-io/badger), [SQLite](https://gitlab.com/cznic/sqlite) and directly on the file system, but I want to also add support for PostgreSQL.

## Features

### Support for C2S ActivityPub:

 * Support for content management activities: `Create`, `Update`, `Delete`.
 * `Follow`, `Accept`, `Reject` with actors as objects.
 * Appreciation activities: `Like`, `Dislike`.
 * Reaction activities: `Block` on actors, `Flag` on objects.
 * Negating content management and appreciation activities using `Undo`.
 * OAuth2 authentication

### Support for S2S ActivityPub

 * Support the same operations as the client to server activities.
 * Capabilities of generating and loading HTTP Signatures from requests.

## Installation

See the [INSTALL](./doc/INSTALL.md) file.

### Running a server in a production environment

If you want to run a FedBOX instance as part of the wider fediverse you must
make sure that it was built with `-tags prod` or `-tags qa` in the build
command because the development builds of FedBOX are not compatible with
public instances due to the fact that the HTTP-Signatures generated are meant
to be replayble from other contexts and lack security.


## Further reading

If you are interested in using FedBOX from an application developer point of view, make sure to read the [Client to Server](./doc/c2s.md) document, which details how the local flavour of ActivityPub C2S API can be used.

More information about FedBOX and the other packages in the GoActivityPub library can be found on the [wiki](https://man.sr.ht/~mariusor/go-activitypub/index.md).

## Contact and feedback

If you have problems, questions, ideas or suggestions, please contact us by posting to the [discussions mailing list](https://lists.sr.ht/~mariusor/go-activitypub-discuss), or on [GitHub](https://github.com/go-ap/fedbox/issues). If you desire quicker feedback, the mailing list is preferred, as the GitHub issues are not checked very often.
