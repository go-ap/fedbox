# FedBOX

FedBOX is a very simple ActivityPub enabled service. Its main purpose is as a reference implementation for the other [go-ap](https://github.com/go-ap) packages.

The secondary purpose is to abstract some common functionality that such a service would use such as HTTP handlers, middlewares, etc.

The current iteration persists data to a postgresql database but I want to add support for a filesystem based method.
