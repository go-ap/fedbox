module github.com/go-ap/fedbox

go 1.25

require (
	git.sr.ht/~mariusor/cache v0.0.0-20250616110250-18a60a6f9473
	git.sr.ht/~mariusor/lw v0.0.0-20250325163623-1639f3fb0e0d
	git.sr.ht/~mariusor/wrapper v0.0.0-20250504120759-5fa47ac25e08
	github.com/alecthomas/kong v1.12.1
	github.com/go-ap/activitypub v0.0.0-20250905102448-e9df599e4528
	github.com/go-ap/auth v0.0.0-20250909130627-7ba95f863264
	github.com/go-ap/cache v0.0.0-20250905102525-152e952d5d86
	github.com/go-ap/client v0.0.0-20250905102653-674f86aded3d
	github.com/go-ap/errors v0.0.0-20250905102357-4480b47a00c4
	github.com/go-ap/filters v0.0.0-20250909130221-883df1a722e4
	github.com/go-ap/jsonld v0.0.0-20250905102310-8480b0fe24d9
	github.com/go-ap/processing v0.0.0-20250909130701-94814c8f7f96
	github.com/go-ap/storage-badger v0.0.0-20250909130821-a62855b57007
	github.com/go-ap/storage-boltdb v0.0.0-20250909130908-e0739c653e6d
	github.com/go-ap/storage-fs v0.0.0-20250909130728-97c1ca4b19c3
	github.com/go-ap/storage-sqlite v0.0.0-20250909130748-597bb3bb03e7
	github.com/go-chi/chi/v5 v5.2.3
	github.com/go-fed/httpsig v1.1.0
	github.com/joho/godotenv v1.5.1
	github.com/pborman/uuid v1.2.1
	golang.org/x/crypto v0.42.0
)

require (
	git.sr.ht/~mariusor/go-xsd-duration v0.0.0-20220703122237-02e73435a078 // indirect
	git.sr.ht/~mariusor/mask v0.0.0-20250114195353-98705a6977b7 // indirect
	git.sr.ht/~mariusor/ssm v0.0.0-20250617180832-c3c0db41a98e // indirect
	github.com/RoaringBitmap/roaring v1.9.4 // indirect
	github.com/bits-and-blooms/bitset v1.24.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgraph-io/badger/v4 v4.8.0 // indirect
	github.com/dgraph-io/ristretto/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/flatbuffers v25.2.10+incompatible // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jdkato/prose v1.2.1 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/mariusor/qstring v0.0.0-20200204164351-5a99d46de39d // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.32 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/openshift/osin v1.0.2-0.20220317075346-0f4d38c6e53f
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/valyala/fastjson v1.6.4 // indirect
	go.etcd.io/bbolt v1.4.3 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	golang.org/x/exp v0.0.0-20250819193227-8b4c13bb791b // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/oauth2 v0.31.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/term v0.35.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
	gopkg.in/neurosnap/sentences.v1 v1.0.7 // indirect
	modernc.org/libc v1.66.8 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.38.2 // indirect
)

replace go.opencensus.io => github.com/census-instrumentation/opencensus-go v0.23.0
