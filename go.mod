module github.com/go-ap/fedbox

go 1.18

require (
	git.sr.ht/~mariusor/lw v0.0.0-20240323171419-d538df4af052
	git.sr.ht/~mariusor/wrapper v0.0.0-20240210113306-c862d947a747
	github.com/go-ap/activitypub v0.0.0-20240408091739-ba76b44c2594
	github.com/go-ap/auth v0.0.0-20240408093715-7b70def65ccb
	github.com/go-ap/cache v0.0.0-20240408093337-846e6272444d
	github.com/go-ap/client v0.0.0-20240408093509-f0721baa55c2
	github.com/go-ap/errors v0.0.0-20240304112515-6077fa9c17b0
	github.com/go-ap/filters v0.0.0-20240408093433-6a48bb3dd711
	github.com/go-ap/jsonld v0.0.0-20221030091449-f2a191312c73
	github.com/go-ap/processing v0.0.0-20240408093907-72bd6bb840b4
	github.com/go-ap/storage-badger v0.0.0-20240401135851-11d962c32b52
	github.com/go-ap/storage-boltdb v0.0.0-20240408095017-242f6a00f38e
	github.com/go-ap/storage-fs v0.0.0-20240408094148-b3bc562af831
	github.com/go-ap/storage-sqlite v0.0.0-20240408094950-793155588873
	github.com/go-chi/chi/v5 v5.0.12
	github.com/go-fed/httpsig v1.1.0
	github.com/joho/godotenv v1.5.1
	github.com/pborman/uuid v1.2.1
	github.com/urfave/cli/v2 v2.25.7
	golang.org/x/crypto v0.22.0
)

require (
	git.sr.ht/~mariusor/mask v0.0.0-20240327084502-ef2a9438457e // indirect
	golang.org/x/oauth2 v0.19.0 // indirect
)

require (
	git.sr.ht/~mariusor/go-xsd-duration v0.0.0-20220703122237-02e73435a078 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/dgraph-io/badger/v4 v4.2.0 // indirect
	github.com/dgraph-io/ristretto v0.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v1.2.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v24.3.25+incompatible // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/klauspost/compress v1.17.7 // indirect
	github.com/mariusor/qstring v0.0.0-20200204164351-5a99d46de39d // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/openshift/osin v1.0.2-0.20220317075346-0f4d38c6e53f
	github.com/pkg/errors v0.9.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/rs/zerolog v1.32.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/valyala/fastjson v1.6.4 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	go.etcd.io/bbolt v1.3.9 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/term v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	modernc.org/gc/v3 v3.0.0-20240304020402-f0dba7c97c2b // indirect
	modernc.org/libc v1.49.2 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
	modernc.org/sqlite v1.29.5 // indirect
	modernc.org/strutil v1.2.0 // indirect
	modernc.org/token v1.1.0 // indirect
)

replace go.opencensus.io => github.com/census-instrumentation/opencensus-go v0.23.0
