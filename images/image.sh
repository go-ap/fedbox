#!/usr/bin/env bash

set -e

_environment=${ENV:-dev}
_hostname=${HOSTNAME:-fedbox}
_listen_port=${PORT:-4000}
_storage=${STORAGE:-all}
_version=${VERSION:-HEAD}

_image_name=${1:-fedbox:"${_environment}-${_storage}"}
_build_name=${2:-localhost/fedbox/builder}

_builder=$(buildah from "${_build_name}":latest)
if [[ -z "${_builder}" ]]; then
    echo "Unable to find builder image: ${_build_name}"
    exit 1
fi

echo "Building image ${_image_name} for host=${_hostname} env:${_environment} storage:${_storage} version:${_version} port:${_listen_port}"

buildah run "${_builder}" make ENV="${_environment}" STORAGE="${_storage}" VERSION="${_version}" all
buildah run "${_builder}" make -C images "${_hostname}.pem"

_image=$(buildah from gcr.io/distroless/static:latest)

buildah config --env ENV="${_environment}" "${_image}"
buildah config --env HOSTNAME="${_hostname}" "${_image}"
buildah config --env LISTEN=:"${_listen_port}" "${_image}"
buildah config --env KEY_PATH=/etc/ssl/certs/${_hostname}.key "${_image}"
buildah config --env CERT_PATH=/etc/ssl/certs/${_hostname}.crt "${_image}"
buildah config --env HTTPS=true "${_image}"
buildah config --env STORAGE="${_storage}" "${_image}"

buildah config --port "${_listen_port}" "${_image}"

buildah config --volume /storage "${_image}"
buildah config --volume /.env "${_image}"

buildah copy --from "${_builder}" "${_image}" /go/src/app/bin/* /bin/
buildah copy --from "${_builder}" "${_image}" /go/src/app/images/*.key /etc/ssl/certs/
buildah copy --from "${_builder}" "${_image}" /go/src/app/images/*.crt /etc/ssl/certs/
buildah copy --from "${_builder}" "${_image}" /go/src/app/images/*.pem /etc/ssl/certs/

buildah config --entrypoint '["/bin/fedbox"]' "${_image}"

# commit
buildah commit "${_image}" "${_image_name}"
