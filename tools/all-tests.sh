#!/bin/bash
set -e
RED='\033[0;31m'
BGREEN='\033[1;32m'
NC='\033[0m' # No Color
run_tests() {
    echo -e "Testing ${BGREEN}${1}${NC}"
    make STORAGE=${1} CGO_ENABLED=0 integration
    make FEDBOX_STORAGE=${1} CGO_ENABLED=0 integration
    make STORAGE=${1} CGO_ENABLED=1 TEST_FLAGS='-race -count=1' integration
    make FEDBOX_STORAGE=${1} CGO_ENABLED=1 TEST_FLAGS='-race -count=1' integration
    echo ""
}

run_tests fs
run_tests sqlite
run_tests boltdb
run_tests badger

find tests/.cache/* -type d -exec rm -rf {} +
