module github.com/go-ap/fedbox

go 1.23

require (
	git.sr.ht/~mariusor/lw v0.0.0-20241117105956-4b4009e28502
	git.sr.ht/~mariusor/wrapper v0.0.0-20240519120935-f877e4d97def
	github.com/go-ap/activitypub v0.0.0-20241123145931-b72f8292c65b
	github.com/go-ap/auth v0.0.0-20241123185744-2de5593b0ca3
	github.com/go-ap/cache v0.0.0-20241123150045-5ee1d68664a4
	github.com/go-ap/client v0.0.0-20241123150455-030e6d4f1054
	github.com/go-ap/errors v0.0.0-20240910140019-1e9d33cc1568
	github.com/go-ap/filters v0.0.0-20241123185521-adc45b30dfb3
	github.com/go-ap/jsonld v0.0.0-20221030091449-f2a191312c73
	github.com/go-ap/processing v0.0.0-20241123185921-294fb9c9b985
	github.com/go-ap/storage-badger v0.0.0-20241123190340-81ee43f03b77
	github.com/go-ap/storage-boltdb v0.0.0-20241123190258-04f18f201344
	github.com/go-ap/storage-fs v0.0.0-20241123190020-01c4b297b443
	github.com/go-ap/storage-sqlite v0.0.0-20241123190202-ea7a72d282c7
	github.com/go-chi/chi/v5 v5.1.0
	github.com/go-fed/httpsig v1.1.0
	github.com/joho/godotenv v1.5.1
	github.com/pborman/uuid v1.2.1
	github.com/urfave/cli/v2 v2.27.5
	golang.org/x/crypto v0.29.0
)

require (
	github.com/RoaringBitmap/roaring v1.9.4 // indirect
	github.com/bits-and-blooms/bitset v1.16.0 // indirect
	github.com/dgraph-io/ristretto/v2 v2.0.0 // indirect
	github.com/jdkato/prose v1.2.1 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	gopkg.in/neurosnap/sentences.v1 v1.0.7 // indirect
)

require (
	git.sr.ht/~mariusor/cache v0.0.0-20241026131931-1ae5432a2760 // indirect
	git.sr.ht/~mariusor/go-xsd-duration v0.0.0-20220703122237-02e73435a078 // indirect
	git.sr.ht/~mariusor/mask v0.0.0-20240327084502-ef2a9438457e // indirect
	git.sr.ht/~mariusor/ssm v0.0.0-20240811085540-34f24cac52b7 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.5 // indirect
	github.com/dgraph-io/badger/v4 v4.4.0 // indirect
	github.com/dgraph-io/ristretto v1.0.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/flatbuffers v24.3.25+incompatible // indirect
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
	golang.org/x/exp v0.0.0-20241108190413-2d47ceb2692f // indirect
	golang.org/x/net v0.31.0 // indirect
	golang.org/x/oauth2 v0.24.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
	golang.org/x/term v0.26.0 // indirect
	golang.org/x/text v0.20.0 // indirect
	google.golang.org/protobuf v1.35.2 // indirect
	modernc.org/gc/v3 v3.0.0-20241004144649-1aea3fae8852 // indirect
	modernc.org/libc v1.61.2 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
	modernc.org/sqlite v1.34.1 // indirect
	modernc.org/strutil v1.2.0 // indirect
	modernc.org/token v1.1.0 // indirect
)

replace go.opencensus.io => github.com/census-instrumentation/opencensus-go v0.23.0
