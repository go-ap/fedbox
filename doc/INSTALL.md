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
$ ./bin/fedboxctl storage bootstrap
```

For a more advanced example, the [`bootstrap.sh`](../tools/bootstrap.sh) script has a more elaborate use case to
automate bootstrapping a project together with adding an Actor and an OAuth2 client.

## Containers

See the [containers](../images/README.md) document for details about how to build podman/docker images or use the ready made ones.
