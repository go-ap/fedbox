make -C images builder

_push() {
    _storage=${1:-all}
    _branch=${GITHUB_REF#"refs/heads/"}
    make -C images STORAGE="${_storage}" VERSION="${_branch}" push
    if [ "${_branch}" == "master" ]; then
        _branch=$(printf "%s-%s" "${_branch}" "$(git rev-parse --short HEAD)")
        make -C images STORAGE="${_storage}" ENV=qa VERSION="${_branch}" push
    fi

    _tag=$(git describe --long --tags || true)
    if [ -n "${_tag}" ]; then
        make -C images STORAGE="${_storage}" ENV=prod VERSION="${_tag}" push
    fi
}

_push
_push fs
_push badger
_push boltdb
_push sqlite

