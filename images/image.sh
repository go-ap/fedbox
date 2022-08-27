#!/usr/bin/env bash

#set -x

_environment=${ENV:-dev}
_hostname=${HOSTNAME:-fedbox}
_listen_port=${PORT:-4000}
_storage=${STORAGE:-all}
_version=${VERSION}

_image_name=${1:-fedbox:"${_environment}-${_storage}"}
_build_name=${2:-localhost/fedbox/builder}

#FROM fedbox/builder AS builder
_builder=$(buildah from "${_build_name}":latest)

if [[ -z ${_builder} ]]; then
    echo "Unable to find builder image: ${_build_name}"
    exit 1
fi

echo "Building image ${_image_name} for host=${_hostname} env:${_environment} storage:${_storage} port:${_listen_port}"

#ARG ENV=dev
#ARG HOSTNAME=fedbox
#ARG STORAGE=all
#ARG VERSION

#ENV GO111MODULE=on
#ENV ENV=${ENV:-dev}
#ENV STORAGE=${STORAGE:-all}
#ENV VERSION=${VERSION:-}

#RUN make ENV=${ENV} STORAGE=${STORAGE} VERSION=${VERSION} all && \
#    docker/gen-certs.sh ${HOSTNAME}
buildah run "${_builder}" make ENV="${ENV:-dev}" STORAGE="${STORAGE:-all}" VERSION="${_version}" all
buildah run "${_builder}" make -C images fedbox.pem

#FROM gcr.io/distroless/static
_image=$(buildah from gcr.io/distroless/static:latest)

#ARG PORT=4000
#ARG ENV=dev
#ARG HOSTNAME=fedbox
#ARG STORAGE=all

#ENV ENV=${ENV:-dev}
buildah config --env ENV="${_environment}" "${_image}"
#ENV STORAGE_PATH=/storage
#buildah config --env STORAGE_PATH=/storage "${_image}
#ENV HOSTNAME="${HOSTNAME:-fedbox}"
buildah config --env HOSTNAME="${_hostname}" "${_image}"
#ENV LISTEN=:${PORT}
buildah config --env LISTEN=:"${_listen_port}" "${_image}"
#ENV KEY_PATH=/etc/ssl/certs/${HOSTNAME}.key
buildah config --env KEY_PATH=/etc/ssl/certs/fedbox.key "${_image}"
#ENV CERT_PATH=/etc/ssl/certs/"${HOSTNAME}.crt
buildah config --env CERT_PATH=/etc/ssl/certs/fedbox.crt "${_image}"
#ENV HTTPS=true
buildah config --env HTTPS=true "${_image}"
#ENV STORAGE=${STORAGE:-all}
buildah config --env STORAGE="${_storage}" "${_image}"

#EXPOSE $PORT
buildah config --port "${_listen_port}" "${_image}"

#VOLUME /storage
buildah config --volume /storage "${_image}"
#VOLUME /.env
buildah config --volume /.env "${_image}"

#COPY --from=builder /go/src/app/bin/* /bin/
buildah copy --from "${_builder}" "${_image}" /go/src/app/bin/* /bin/
#COPY --from=builder /go/src/app/*.key /go/src/app/*.crt /go/src/app/*.pem /etc/ssl/certs/
buildah copy --from "${_builder}" "${_image}" /go/src/app/images/fedbox.key /etc/ssl/certs/
buildah copy --from "${_builder}" "${_image}" /go/src/app/images/fedbox.crt /etc/ssl/certs/
buildah copy --from "${_builder}" "${_image}" /go/src/app/images/fedbox.pem /etc/ssl/certs/

#CMD ["/bin/fedbox"]
buildah config --entrypoint '["/bin/fedbox"]' "${_image}"

# commit
buildah commit "${_image}" "${_image_name}"
