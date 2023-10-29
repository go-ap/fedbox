#!/bin/bash
set -e

run_tests() {
  echo "Testing ${1}"
    make STORAGE=${1} CGO_ENABLED=0 integration
    make FEDBOX_STORAGE=${1} CGO_ENABLED=0 integration
    make STORAGE=${1} CGO_ENABLED=1 TEST_FLAGS='-race -count=1' integration
    make FEDBOX_STORAGE=${1} CGO_ENABLED=1 TEST_FLAGS='-race -count=1' integration
}

run_tests fs
run_tests sqlite
run_tests boltdb
run_tests badger

find tests/.cache/* -type d -print -delete
