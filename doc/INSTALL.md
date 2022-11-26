## Getting the source

```sh
$ git clone https://github.com/go-ap/fedbox
$ cd fedbox
```

## Compiling

```sh
$ make all
```

## Editing the configuration

```sh
$ cp .env.dist .env
$ $EDITOR .env
```

## Bootstrapping

```sh
$ ./bin/fedboxctl bootstrap

# add an admin account
$ ./bin/fedboxctl ap actor add admin
admin's pw:
pw again:

# add an OAuth2 client for interacting with the server
$ ./bin/fedboxctl oauth client add --redirectUri http://example.com/callback
client's pw:
pw again:
```

## Containers

See the [containers](./containers.md) document for details about podman for running the server.
