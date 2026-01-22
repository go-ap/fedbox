#!/bin/sh

set -ex

_action=${1:-build} # or push
_storage=${2:-all}

make -C images builder

_sha=$(git rev-parse --short HEAD)
_branch=$(git branch --points-at=${_sha} | tail -n1 | tr -d '* ')

_version=$(printf "%s-%s" "${_branch}" "${_sha}")

make -C images STORAGE="${_storage}" ENV=dev VERSION="${_version}" "${_action}"

if [ "${_branch}" = "master" ]; then
    make -C images STORAGE="${_storage}" ENV=qa VERSION="${_version}" "${_action}"
fi

_tag=$(git describe --long --tags || true)
if [ -n "${_tag}" ]; then
  make -C images STORAGE="${_storage}" ENV=prod VERSION="${_tag}" "${_action}"
fi
