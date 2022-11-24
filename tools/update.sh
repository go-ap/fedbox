#!/bin/bash
set -xe

deps=(activitypub auth client errors jsonld processing storage-fs storage-sqlite storage-boltdb storage-badger)

for dep in ${deps[@]}; do
    sha=$(git --git-dir="../go-ap/${dep}/.git" log -n1 --format=tformat:%h)
    go get -u github.com/go-ap/${dep}@${sha}
done
if [[ -d "../wrapper" ]]; then
    sha=$(git --git-dir="../wrapper/.git" log -n1 --format=tformat:%h)
    go get -u git.sr.ht/~mariusor/wrapper@${sha}
fi
deps=(render)

for dep in ${deps[@]}; do
    sha=$(git --git-dir="../${dep}/.git" log -n1 --format=tformat:%h)
    go get -u github.com/mariusor/${dep}@${sha}
done
go mod tidy

make test

set +e
#ake STORAGE=fs integration
#ake STORAGE=boltdb integration
#ake STORAGE=badger integration
#ake STORAGE=sqlite integration
