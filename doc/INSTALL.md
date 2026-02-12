## Getting the source

```sh
$ git clone https://github.com/go-ap/fedbox
$ cd fedbox
```

## Compiling

```sh
$ make all
```

Compiling for a specific storage backend:

```shell
$ make STORAGE=sqlite all
```

Compiling for the production environment:

```shell
$ make ENV=prod all
```

## Editing the configuration

```sh
$ cp .env.dist .env
$ $EDITOR .env
```

## Bootstrapping

This step ensures that the storage method we're using gets initialized.

```sh
$ ./bin/fedbox storage bootstrap
```

For a more advanced example, the [`bootstrap.sh`](../tools/bootstrap.sh) script has a more elaborate use case to
automate bootstrapping a project together with adding an Actor and an OAuth2 client.

## Authorization

FedBOX is strictly an ActivityPub server, so despite the fact that it advertises OAuth2 authorization endpoints
for its actors, it does not provide them.

In order to make use of that functionality and be able to go through an authorization flow, you need to have running
the [authorization microservice](https://github.com/go-ap/authorize#authorization-handlers-on-top-of-go-activitypub-storage) alongside it.

It should be able to run using the same configuration `.env` file as FedBOX.

This adds the additional requirement that the requests done to the OAuth2 authorization endpoints are proxied to the correct service.

Here's an example for caddy:

```caddyfile
# actors oauth endpoints
handle /actors/*/oauth/* {
	reverse_proxy http://authorize-service:1234 {
		transport http
	}
}
# root service oauth endpoints
handle /oauth/* {
	reverse_proxy http://authorize-service:1234 {
		transport http
	}
}
```

## WebFinger actor discovery

Similarly to how OAuth2 is handled by a different service that uses the same storage backend as FedBOX, for providing
web-finger we have a different [microservice for .well-known](https://github.com/go-ap/webfinger?tab=readme-ov-file#webfinger-handlers-on-top-of-go-activitypub-storage) end-points.

It can also be run with the same configuration `.env` file as FedBOX.

Similarly to above, it needs additional request proxying setup:
```caddyfile
handle /.well-known/* {
	reverse_proxy http://well-known-service:4555 {
		transport http
	}
}
```

## Containers

See the [containers](../images/README.md) document for details about how to build podman/docker images or use the existing ones.
