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

This step ensures that the storage method we're using gets initialized.

```sh
$ ./bin/fedboxctl bootstrap
```

## Containers

See the [containers](./containers.md) document for details about podman for running the server.
