#!/bin/bash
set -e

echo "testing fs"
make FEDBOX_STORAGE=fs integration
echo "testing boltdb"
make FEDBOX_STORAGE=boltdb integration
echo "testing badger"
make FEDBOX_STORAGE=badger integration
echo "testing sqlite"
make FEDBOX_STORAGE=sqlite integration

export TEST_FLAGS='-race -count=1'

make CGO_ENABLED=1 STORAGE=fs integration
make CGO_ENABLED=1 STORAGE=boltdb integration
make CGO_ENABLED=1 STORAGE=badger integration

unset TEST_FLAGS
export CGO_ENABLED=0
make STORAGE=sqlite integration

