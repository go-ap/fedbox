make -C images builder
_storage=${1:-all}
_push=${2:-build}

_branch=$(git branch --show-current)
make -C images STORAGE="${_storage}" VERSION="${_branch}" push
if [ "${_branch}" == "master" ]; then
    _branch=$(printf "%s-%s" "${_branch}" "$(git rev-parse --short HEAD)")
    make -C images STORAGE="${_storage}" ENV=qa VERSION="${_branch}" ${_push}
fi

_tag=$(git describe --long --tags || true)
if [ -n "${_tag}" ]; then
    make -C images STORAGE="${_storage}" ENV=prod VERSION="${_tag}" ${_push}
fi

