package app

import (
	"context"
	ap "github.com/go-ap/handlers"
	st "github.com/go-ap/storage"
)

type CtxtKey string

var OwnerKey = CtxtKey("__owner")
var ActorKey = CtxtKey("__actor")

func Loader(ctx context.Context) (st.ActorLoader, bool) {
	ctxVal := ctx.Value(ap.RepositoryKey)
	s, ok := ctxVal.(st.ActorLoader)
	return s, ok
}
