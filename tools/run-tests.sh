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
    run_tests fs
    run_tests sqlite
    run_tests boltdb
    run_tests badger
else
    run_tests "${1}"
fi

find tests/.cache/* -type d -exec rm -rf {} +
