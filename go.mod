module github.com/go-ap/fedbox

go 1.18

require (
	git.sr.ht/~mariusor/lw v0.0.0-20230317075520-07e173563bf8
	git.sr.ht/~mariusor/wrapper v0.0.0-20230104101555-9bfc303f6588
	github.com/go-ap/activitypub v0.0.0-20230509081506-c63baa5ec992
	github.com/go-ap/auth v0.0.0-20230509081727-89fa37abf2fa
	github.com/go-ap/client v0.0.0-20230509081643-66f4b336aaef
	github.com/go-ap/errors v0.0.0-20221205040414-01c1adfc98ea
	github.com/go-ap/filters v0.0.0-20230509081808-5c9e638a1c6d
	github.com/go-ap/jsonld v0.0.0-20221030091449-f2a191312c73
	github.com/go-ap/processing v0.0.0-20230509081745-6ee86e9025cc
	github.com/go-ap/storage-badger v0.0.0-20230509082039-5e42f26330a6
	github.com/go-ap/storage-boltdb v0.0.0-20230509082016-82d5c953bad9
	github.com/go-ap/storage-fs v0.0.0-20230509081835-76326a325f7b
	github.com/go-ap/storage-sqlite v0.0.0-20230509081951-e0101e085d49
	github.com/go-chi/chi/v5 v5.0.7
	github.com/go-fed/httpsig v1.1.0
	github.com/joho/godotenv v1.4.0
	github.com/mariusor/qstring v0.0.0-20200204164351-5a99d46de39d
	github.com/mariusor/render v1.5.1-0.20221026090743-ab78c1b3aa95
	github.com/openshift/osin v1.0.1
	github.com/pborman/uuid v1.2.1
	github.com/urfave/cli/v2 v2.3.0
	golang.org/x/crypto v0.9.0
	golang.org/x/oauth2 v0.8.0
)

require (
	git.sr.ht/~mariusor/go-xsd-duration v0.0.0-20220703122237-02e73435a078 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.1 // indirect
	github.com/dgraph-io/badger/v3 v3.2103.5 // indirect
	github.com/dgraph-io/ristretto v0.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-chi/chi v4.1.2+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v1.1.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v23.3.3+incompatible // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/klauspost/compress v1.16.5 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.18 // indirect
	github.com/mattn/go-sqlite3 v1.14.16 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/rs/zerolog v1.29.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/valyala/fastjson v1.6.4 // indirect
	go.etcd.io/bbolt v1.3.7 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/mod v0.10.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
	golang.org/x/term v0.8.0 // indirect
	golang.org/x/tools v0.9.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	lukechampine.com/uint128 v1.3.0 // indirect
	modernc.org/cc/v3 v3.40.0 // indirect
	modernc.org/ccgo/v3 v3.16.13 // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/opt v0.1.3 // indirect
	modernc.org/sqlite v1.22.1 // indirect
	modernc.org/strutil v1.1.3 // indirect
	modernc.org/token v1.1.0 // indirect
)

replace go.opencensus.io => github.com/census-instrumentation/opencensus-go v0.23.0
