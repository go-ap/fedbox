#!/bin/bash
set -e
RED='\033[0;31m'
BGREEN='\033[1;32m'
NC='\033[0m' # No Color

run_tests() {
    _storage="${1}"
    echo -e "Testing ${BGREEN}${_storage}${NC}"
    make STORAGE=${_storage} CGO_ENABLED=0 integration
    make FEDBOX_STORAGE=${_storage} CGO_ENABLED=0 integration
    make STORAGE=${_storage} CGO_ENABLED=1 TEST_FLAGS='-race -count=1' integration
    make FEDBOX_STORAGE=${_storage} CGO_ENABLED=1 TEST_FLAGS='-race -count=1' integration
    echo ""
}

if [[ "${1}" = "" ]]; then
    _tests=(fs sqlite boltdb badger)
    _tests=(fs sqlite boltdb)
else
    _tests=${@}
fi

for _test in ${_tests[@]} ; do
    run_tests "${_test}"
done

find ./tests/.cache/ -mindepth 1 -type d -exec rm -rf {} +
