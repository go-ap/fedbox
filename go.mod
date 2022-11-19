module github.com/go-ap/fedbox

go 1.18

require (
	git.sr.ht/~mariusor/lw v0.0.0-20221102084913-34fb8eae441e
	git.sr.ht/~mariusor/wrapper v0.0.0-20211204195804-3033a1099e0f
	github.com/dgraph-io/badger/v3 v3.2103.2
	github.com/go-ap/activitypub v0.0.0-20221119120906-cb8207231e18
	github.com/go-ap/auth v0.0.0-20221119121244-66f93d52b757
	github.com/go-ap/client v0.0.0-20221119121059-9d87174a90b2
	github.com/go-ap/errors v0.0.0-20221115052505-8aaa26f930b4
	github.com/go-ap/httpsig v0.0.0-20210714162115-62a09257db51
	github.com/go-ap/jsonld v0.0.0-20221030091449-f2a191312c73
	github.com/go-ap/processing v0.0.0-20221119121403-a6040a70fd46
	github.com/go-chi/chi/v5 v5.0.7
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/joho/godotenv v1.4.0
	github.com/mariusor/qstring v0.0.0-20200204164351-5a99d46de39d
	github.com/mariusor/render v1.5.1-0.20221026090743-ab78c1b3aa95
	github.com/mattn/go-sqlite3 v1.14.15
	github.com/openshift/osin v1.0.1
	github.com/pborman/uuid v1.2.1
	github.com/urfave/cli/v2 v2.3.0
	go.etcd.io/bbolt v1.3.6
	golang.org/x/crypto v0.3.0
	golang.org/x/oauth2 v0.2.0
	modernc.org/sqlite v1.19.1
)

require (
	git.sr.ht/~mariusor/go-xsd-duration v0.0.0-20220703122237-02e73435a078 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.1 // indirect
	github.com/dgraph-io/ristretto v0.1.1 // indirect
	github.com/dgryski/go-farm v0.0.0-20200201041132-a6ae2369ad13 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/go-chi/chi v4.1.2+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v1.0.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v22.9.29+incompatible // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/klauspost/compress v1.15.11 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20220927061507-ef77025ab5aa // indirect
	github.com/rs/xid v1.4.0 // indirect
	github.com/rs/zerolog v1.28.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/valyala/fastjson v1.6.3 // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4 // indirect
	golang.org/x/net v0.2.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
	golang.org/x/term v0.2.0 // indirect
	golang.org/x/text v0.4.0 // indirect
	golang.org/x/tools v0.1.12 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	lukechampine.com/uint128 v1.2.0 // indirect
	modernc.org/cc/v3 v3.40.0 // indirect
	modernc.org/ccgo/v3 v3.16.9 // indirect
	modernc.org/libc v1.20.2 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.4.0 // indirect
	modernc.org/opt v0.1.3 // indirect
	modernc.org/strutil v1.1.3 // indirect
	modernc.org/token v1.0.1 // indirect
)

replace go.opencensus.io => github.com/census-instrumentation/opencensus-go v0.23.0
