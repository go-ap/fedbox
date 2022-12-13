#!/bin/sh

set -ex

make -C images builder
_storage=${1:-all}
_push=${2:-build}

_sha=$(git rev-parse --short HEAD)
_branch=$(git branch --show-current)
make -C images STORAGE="${_storage}" ENV=dev VERSION="${_branch}" ${_push}
if [ "${_branch}" = "master" ]; then
    _branch=$(printf "%s-%s" "${_branch}" "${_sha}")
    make -C images STORAGE="${_storage}" ENV=qa VERSION="${_branch}" ${_push}
fi

_tag=$(git describe --long --tags || true)
if [ -n "${_tag}" ]; then
    make -C images STORAGE="${_storage}" ENV=prod VERSION="${_tag}" ${_push}
fi

