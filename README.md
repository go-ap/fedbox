# FedBOX

[![MIT Licensed](https://img.shields.io/github/license/go-ap/fedbox.svg)](https://raw.githubusercontent.com/go-ap/fedbox/master/LICENSE)
[![Build Status](https://builds.sr.ht/~mariusor/fedbox.svg)](https://builds.sr.ht/~mariusor/fedbox)
[![Test Coverage](https://codecov.io/gh/go-ap/fedbox/branch/master/graph/badge.svg)](https://codecov.io/gh/go-ap/fedbox)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/29664f7ae6c643bca76700143e912cd3)](https://www.codacy.com/app/go-ap/fedbox/dashboard)

FedBOX is a very simple ActivityPub enabled service. Its main purpose is as a reference implementation for the other [go-ap](https://github.com/go-ap) packages.

The secondary purpose is to abstract some of the common functionality that such a service would use, such as: HTTP handlers and middlewares, storage and filtering etc.

The current iteration can persist data to PostgreSQL and BoltDB, but I want to also add support for a filesystem based method.
