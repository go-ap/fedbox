package app

import (
	"context"
	st "github.com/go-ap/fedbox/storage"
	ap "github.com/go-ap/handlers"
)

type CtxtKey string

var OwnerKey = CtxtKey("__owner")
var ActorKey = CtxtKey("__actor")

func Loader(ctx context.Context) (st.ActorLoader, bool) {
	ctxVal := ctx.Value(ap.RepositoryKey)
	s, ok := ctxVal.(st.ActorLoader)
	return s, ok
}
