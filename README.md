# FedBOX

FedBOX is a very simple ActivityPub enabled service. Its main purpose is as a reference implementation for the other [go-ap](https://github.com/go-ap) packages.

The secondary purpose is to abstract some of the common functionality that such a service would use, such as: HTTP handlers and middlewares, storage and filtering etc.

The current iteration persists data to PostgreSQL and BoltDB, but I want to also add support for a filesystem based method.
