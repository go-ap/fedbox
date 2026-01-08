#!/usr/bin/env bash

set -e
_context=$(realpath "../")

_environment=${ENV:-dev}
_hostname=${FEDBOX_HOSTNAME:-fedbox}
_listen_http_port=${HTTP_PORT:-4000}
_listen_ssh_port=${PORT:-4022}
_storage=${STORAGE:-all}
_version=${VERSION:-HEAD}

_image_name=${1:-"${_hostname}:${_environment}-${_storage}"}

HOST_GOCACHE=$(go env GOCACHE)
HOST_GOMODCACHE=$(go env GOMODCACHE)

GOCACHE=/root/.cache/go-build
GOMODCACHE=/go/pkg/mod

_builder=$(buildah from docker.io/library/golang:1.25-alpine)

buildah run "${_builder}" /sbin/apk update
buildah run "${_builder}" /sbin/apk add make bash openssl upx

buildah config --env GO111MODULE=on "${_builder}"
buildah config --env GOWORK=off "${_builder}"
buildah config --env "GOCACHE=${GOCACHE}" "${_builder}"
buildah config --env "GOMODCACHE=${GOMODCACHE}" "${_builder}"

buildah config --workingdir /go/src/app "${_builder}"

echo "Building image ${_image_name} for host=${_hostname} env:${_environment} storage:${_storage} version:${_version} port:${_listen_http_port}"

buildah run \
    --mount="type=bind,rw,source=${HOST_GOCACHE},destination=${GOCACHE}" \
    --mount="type=bind,rw,source=${HOST_GOMODCACHE},destination=${GOMODCACHE}" \
    --mount="type=bind,rw,source=${_context},destination=/go/src/app" \
    --mount=type=cache,rw,id=bin,target=/go/src/app/bin "${_builder}" \
    make ENV="${_environment}" STORAGE="${_storage}" VERSION="${_version}" FEDBOX_HOSTNAME="${_hostname}" clean all cert

# copy binaries from cache to builder container fs
buildah run \
    --mount=type=cache,rw,id=bin,target=/tmp/bin "${_builder}" \
    cp -ri /tmp/bin /go/src/

_image=$(buildah from gcr.io/distroless/static:latest)

buildah config --env "ENV=${_environment}" "${_image}"
buildah config --env "HOSTNAME=${_hostname}" "${_image}"
buildah config --env "HTTP_PORT=${_listen_http_port}" "${_image}"
buildah config --env "SSH_PORT=${_listen_ssh_port}" "${_image}"
buildah config --env "KEY_PATH=/etc/ssl/certs/${_hostname}.key" "${_image}"
buildah config --env "CERT_PATH=/etc/ssl/certs/${_hostname}.crt" "${_image}"
buildah config --env "STORAGE=${_storage}" "${_image}"
buildah config --env HTTPS=true "${_image}"

buildah config --port "${_listen_http_port}" "${_image}"
buildah config --port "${_listen_ssh_port}" "${_image}"

buildah config --volume /storage "${_image}"
buildah config --volume /.env "${_image}"

buildah copy --from "${_builder}" "${_image}" /go/src/bin/* /bin/
buildah copy --from "${_builder}" "${_image}" "/go/src/bin/${_hostname}.key" /etc/ssl/certs/
buildah copy --from "${_builder}" "${_image}" "/go/src/bin/${_hostname}.crt" /etc/ssl/certs/
buildah copy --from "${_builder}" "${_image}" "/go/src/bin/${_hostname}.pem" /etc/ssl/certs/

buildah config --entrypoint '["/bin/fedbox"]' "${_image}"

# commit
buildah commit "${_image}" "${_image_name}"
