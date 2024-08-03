module github.com/go-ap/fedbox

go 1.21

require (
	git.sr.ht/~mariusor/lw v0.0.0-20240323171419-d538df4af052
	git.sr.ht/~mariusor/wrapper v0.0.0-20240519120935-f877e4d97def
	github.com/go-ap/activitypub v0.0.0-20240408091739-ba76b44c2594
	github.com/go-ap/auth v0.0.0-20240803164334-84c5e7cf500d
	github.com/go-ap/cache v0.0.0-20240408093337-846e6272444d
	github.com/go-ap/client v0.0.0-20240803164250-72674fd63382
	github.com/go-ap/errors v0.0.0-20240304112515-6077fa9c17b0
	github.com/go-ap/filters v0.0.0-20240801112128-c16e26a892c4
	github.com/go-ap/jsonld v0.0.0-20221030091449-f2a191312c73
	github.com/go-ap/processing v0.0.0-20240803123347-9d6c8c8868f2
	github.com/go-ap/storage-badger v0.0.0-20240801114622-4d730a7aea87
	github.com/go-ap/storage-boltdb v0.0.0-20240803123915-a133721967e9
	github.com/go-ap/storage-fs v0.0.0-20240803123436-43fd4521ee5a
	github.com/go-ap/storage-sqlite v0.0.0-20240803123516-e64a31bf814d
	github.com/go-chi/chi/v5 v5.1.0
	github.com/go-fed/httpsig v1.1.0
	github.com/joho/godotenv v1.5.1
	github.com/pborman/uuid v1.2.1
	github.com/urfave/cli/v2 v2.27.3
	golang.org/x/crypto v0.25.0
)

require (
	git.sr.ht/~mariusor/cache v0.0.0-20240803161421-766939937a58 // indirect
	git.sr.ht/~mariusor/mask v0.0.0-20240327084502-ef2a9438457e // indirect
	golang.org/x/oauth2 v0.21.0 // indirect
)

require (
	git.sr.ht/~mariusor/go-xsd-duration v0.0.0-20220703122237-02e73435a078 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/dgraph-io/badger/v4 v4.2.0 // indirect
	github.com/dgraph-io/ristretto v0.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v24.3.25+incompatible // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/mariusor/qstring v0.0.0-20200204164351-5a99d46de39d // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/openshift/osin v1.0.2-0.20220317075346-0f4d38c6e53f
	github.com/pkg/errors v0.9.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/rs/zerolog v1.33.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/valyala/fastjson v1.6.4 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	go.etcd.io/bbolt v1.3.10 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/term v0.22.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	modernc.org/gc/v3 v3.0.0-20240801135723-a856999a2e4a // indirect
	modernc.org/libc v1.56.0 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
	modernc.org/sqlite v1.31.1 // indirect
	modernc.org/strutil v1.2.0 // indirect
	modernc.org/token v1.1.0 // indirect
)

replace go.opencensus.io => github.com/census-instrumentation/opencensus-go v0.23.0
