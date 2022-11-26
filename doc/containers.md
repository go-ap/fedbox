## Container images

We are building podman[^1] container images for FedBOX that can be found at [quay.io](quay.io/go-ap/fedbox).

The `tools/run-container` script can be used as an example of how to run such a container.

```sh
# /var/cache/fedbox must exist and be writable as current user
# /var/cache/fedbox/env must be a valid env file as shown in the installation section
$ podman run --network=host --name=FedBOX -v /var/cache/fedbox/env:/.env -v /var/cache/fedbox:/storage --env-file=/var/cache/fedbox/env quay.io/go-ap/fedbox:latest
```

### Running fedboxctl commands in the containers

```sh
# running in the same context as above
$ podman exec --env-file=/var/cache/fedbox/env FedBOX bootstrap
```

[^1] And docker, of course.

