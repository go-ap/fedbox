module github.com/go-ap/fedbox

go 1.21

require (
	git.sr.ht/~mariusor/lw v0.0.0-20230317075520-07e173563bf8
	git.sr.ht/~mariusor/wrapper v0.0.0-20230710102058-fc38877da4fe
	github.com/go-ap/activitypub v0.0.0-20231114162308-e219254dc5c9
	github.com/go-ap/auth v0.0.0-20231114164013-a1640365bf33
	github.com/go-ap/cache v0.0.0-20231114162417-36177bcbd4a9
	github.com/go-ap/client v0.0.0-20231114162455-f09cf9766e95
	github.com/go-ap/errors v0.0.0-20231003111023-183eef4b31b7
	github.com/go-ap/filters v0.0.0-20231114163756-0a70c1a4a942
	github.com/go-ap/jsonld v0.0.0-20221030091449-f2a191312c73
	github.com/go-ap/processing v0.0.0-20231114164044-596105c0aac5
	github.com/go-ap/storage-badger v0.0.0-20231114164254-83bfd520a801
	github.com/go-ap/storage-boltdb v0.0.0-20231114164236-78b8f85c6fda
	github.com/go-ap/storage-fs v0.0.0-20231224140711-c4e8e4fe02d1
	github.com/go-ap/storage-sqlite v0.0.0-20231112181059-f32529430fb8
	github.com/go-chi/chi/v5 v5.0.11
	github.com/go-fed/httpsig v1.1.0
	github.com/joho/godotenv v1.5.1
	github.com/mariusor/render v1.5.1-0.20221026090743-ab78c1b3aa95
	github.com/pborman/uuid v1.2.1
	github.com/urfave/cli/v2 v2.25.7
	golang.org/x/crypto v0.17.0
	golang.org/x/oauth2 v0.15.0
)

require github.com/mariusor/qstring v0.0.0-20200204164351-5a99d46de39d // indirect

require (
	git.sr.ht/~mariusor/go-xsd-duration v0.0.0-20220703122237-02e73435a078 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/dgraph-io/badger/v4 v4.2.0 // indirect
	github.com/dgraph-io/ristretto v0.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-chi/chi v4.1.2+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v1.2.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v23.5.26+incompatible // indirect
	github.com/google/uuid v1.5.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.19 // indirect
	github.com/openshift/osin v1.0.2-0.20220317075346-0f4d38c6e53f
	github.com/pkg/errors v0.9.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/rs/zerolog v1.31.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/valyala/fastjson v1.6.4 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	go.etcd.io/bbolt v1.3.8 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/mod v0.14.0 // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/term v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.16.1 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
	lukechampine.com/uint128 v1.3.0 // indirect
	modernc.org/cc/v3 v3.41.0 // indirect
	modernc.org/ccgo/v3 v3.16.15 // indirect
	modernc.org/libc v1.38.0 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.7.2 // indirect
	modernc.org/opt v0.1.3 // indirect
	modernc.org/sqlite v1.28.0 // indirect
	modernc.org/strutil v1.2.0 // indirect
	modernc.org/token v1.1.0 // indirect
)

replace go.opencensus.io => github.com/census-instrumentation/opencensus-go v0.23.0
