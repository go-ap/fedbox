module github.com/go-ap/fedbox

go 1.23

require (
	git.sr.ht/~mariusor/lw v0.0.0-20241117105956-4b4009e28502
	git.sr.ht/~mariusor/wrapper v0.0.0-20240519120935-f877e4d97def
	github.com/go-ap/activitypub v0.0.0-20241228090954-75890bd9cfda
	github.com/go-ap/auth v0.0.0-20241228091549-42783d36aae4
	github.com/go-ap/cache v0.0.0-20241228091143-603fda798574
	github.com/go-ap/client v0.0.0-20241228091406-581647f214a8
	github.com/go-ap/errors v0.0.0-20241212155021-5a598b6bf467
	github.com/go-ap/filters v0.0.0-20241228091325-b92f78983bb6
	github.com/go-ap/jsonld v0.0.0-20221030091449-f2a191312c73
	github.com/go-ap/processing v0.0.0-20241228091655-8f33f4bd24e0
	github.com/go-ap/storage-badger v0.0.0-20241216210132-0b04fa45ce83
	github.com/go-ap/storage-boltdb v0.0.0-20241228091953-41a5bc0deb11
	github.com/go-ap/storage-fs v0.0.0-20241228091737-60d5cc386a81
	github.com/go-ap/storage-sqlite v0.0.0-20241228092047-a0764404cfc5
	github.com/go-chi/chi/v5 v5.2.0
	github.com/go-fed/httpsig v1.1.0
	github.com/joho/godotenv v1.5.1
	github.com/pborman/uuid v1.2.1
	github.com/urfave/cli/v2 v2.27.5
	golang.org/x/crypto v0.31.0
)

require (
	github.com/RoaringBitmap/roaring v1.9.4 // indirect
	github.com/bits-and-blooms/bitset v1.20.0 // indirect
	github.com/dgraph-io/ristretto/v2 v2.0.1 // indirect
	github.com/jdkato/prose v1.2.1 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	gopkg.in/neurosnap/sentences.v1 v1.0.7 // indirect
)

require (
	git.sr.ht/~mariusor/cache v0.0.0-20241212172633-e1563652acb4
	git.sr.ht/~mariusor/go-xsd-duration v0.0.0-20220703122237-02e73435a078 // indirect
	git.sr.ht/~mariusor/mask v0.0.0-20240327084502-ef2a9438457e // indirect
	git.sr.ht/~mariusor/ssm v0.0.0-20241220163816-32d18afe7b22 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.5 // indirect
	github.com/dgraph-io/badger/v4 v4.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/google/flatbuffers v24.12.23+incompatible // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/mariusor/qstring v0.0.0-20200204164351-5a99d46de39d // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.24 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/openshift/osin v1.0.2-0.20220317075346-0f4d38c6e53f
	github.com/pkg/errors v0.9.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/rs/zerolog v1.33.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/valyala/fastjson v1.6.4 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	go.etcd.io/bbolt v1.3.11 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/exp v0.0.0-20241217172543-b2144cdd0a67 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/oauth2 v0.24.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/term v0.27.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/protobuf v1.36.1 // indirect
	modernc.org/gc/v3 v3.0.0-20241223112719-96e2e1e4408d // indirect
	modernc.org/libc v1.61.5 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.8.0 // indirect
	modernc.org/sqlite v1.34.4 // indirect
	modernc.org/strutil v1.2.0 // indirect
	modernc.org/token v1.1.0 // indirect
)

replace go.opencensus.io => github.com/census-instrumentation/opencensus-go v0.23.0
