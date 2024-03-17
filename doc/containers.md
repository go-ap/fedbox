## Container images

We are building podman[^1] container images for FedBOX that can be found at [quay.io](https://quay.io/go-ap/fedbox).

The containers are split onto two dimensions: 
* run environment type: `prod`, `qa` or `dev`:
  * `dev` images are built with all debugging information and logging context possible, built for every push.
  * `qa` images are built by stripping the binaries of unneeded elements. Less debugging and logging possible,
also built every push.
  * `prod` images are similar to `qa` ones but are created only when a tagged version of the project is released.
* storage type: `fs`, `sqlite`, `boltdb`, or all. 
  * `fs` the JSON-Ld documents are saved verbatim on disk in a tree folder structure. Fast and error prone.
  * `sqlite`: the JSON-Ld documents are saved in a key-value store under the guise of a database table. 
Querying large collections could be slow.
  * `boltdb`: a more traditional key-value store in the Go ecosystem. 

A resulting image tag has information about all of these, and it would look like `qa-boltdb` 
(stripped image supporting boltdb storage), or `dev` (not stripped image with all storage options available).

The `tools/run-container` script can be used as an example of how to run such a container.

```sh
# /var/cache/fedbox must exist and be writable as current user
# /var/cache/fedbox/env must be a valid env file as shown in the INSTALL document.
$ podman run --network=host --name=FedBOX -v /var/cache/fedbox/env:/.env -v /var/cache/fedbox:/storage --env-file=/var/cache/fedbox/env quay.io/go-ap/fedbox:latest
```

### Running *ctl commands in the containers

```sh
# running with the same configuration environment as above
$ podman exec --env-file=/var/cache/fedbox/env FedBOX fedboxctl bootstrap
$ podman exec --env-file=/var/cache/fedbox/env FedBOX fedboxctl pub actor add --type Application
Enter the actor's name: test
test's pw: 
pw again: 
Added "Application" [test]: https://fedbox/actors/22200000-0000-0000-0001-93e066611fcb
```

[^1] And docker, of course.

