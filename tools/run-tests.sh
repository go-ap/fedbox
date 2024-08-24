#!/bin/bash
set -e
RED='\033[0;31m'
GREEN='\033[1;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

run_tests() {
    _storage=${1}
    echo -e "Testing ${GREEN}${_storage}${NC} with CGO ${YELLOW}Disabled${NC}"
    make        STORAGE="${_storage}" CGO_ENABLED=0 TEST_FLAGS='-count=1 -cover' integration
    make FEDBOX_STORAGE="${_storage}" CGO_ENABLED=0 TEST_FLAGS='-count=1 -cover' integration
    echo -e "Testing ${GREEN}${_storage}${NC} with CGO ${YELLOW}Enabled${NC}"
    make        STORAGE="${_storage}" CGO_ENABLED=1 TEST_FLAGS='-race -count=1' integration
    make FEDBOX_STORAGE="${_storage}" CGO_ENABLED=1 TEST_FLAGS='-race -count=1' integration
    echo ""
}

if [[ "${1}" = "" ]]; then
    #_tests=(fs sqlite boltdb badger)
    _tests=(fs sqlite boltdb)
else
    _tests="${@}"
fi

for _test in ${_tests[@]} ; do
    run_tests "${_test}"
done

find ./tests/.cache/ -mindepth 1 -type d -exec rm -rf {} +
