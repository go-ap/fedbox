## getting the source

```sh
$ git clone https://github.com/go-ap/fedbox
$ cd fedbox
```

## compiling 

```sh
$ make all
```

## editing the configuration 

```sh
$ cp .env.example .env
$ $EDITOR .env
```

## bootstrapping

```sh
$ ./bin/ctl bootstrap

$ ./bin/ctl actor add admin
admin's pw: 
pw again: 

$ ./bin/ctl oauth client add --redirectUri http://example.com/callback
client's pw:
pw again:
```
