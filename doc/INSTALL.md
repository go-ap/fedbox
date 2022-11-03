## getting the source

```sh
$ git clone https://github.com/go-ap/fedbox
$ cd fedbox
```

## getting the dependencies

```sh
$ go get .
$ go get github.com/go-ap/fedbox/internal/cmd
```

## compiling

```sh
$ make all
```

## editing the configuration

```sh
$ cp .env.dist .env
$ $EDITOR .env
```

## bootstrapping

```sh
$ ./bin/fedboxctl bootstrap

$ ./bin/fedboxctl ap actor add admin
admin's pw:
pw again:

$ ./bin/fedboxctl oauth client add --redirectUri http://example.com/callback
client's pw:
pw again:
```
