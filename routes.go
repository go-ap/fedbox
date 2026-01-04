package fedbox

import (
	"net/http"
	"strings"
	"time"

	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/errors"
	"github.com/go-ap/processing"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type justPrintLogger func(string, ...any)

func (c justPrintLogger) Printf(f string, v ...interface{}) {
	c(strings.TrimSpace(f), v...)
}

func checkOriginForBlockedActors(r *http.Request, origin string) bool {
	return true
}

func (f *FedBOX) Routes() func(chi.Router) {
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		AllowOriginFunc:  checkOriginForBlockedActors,
		MaxAge:           int(time.Hour.Seconds()),
		Debug:            !f.Conf.Env.IsProd(),
	})
	c.Log = justPrintLogger(f.Logger.WithContext(lw.Ctx{"log": "cors"}).Tracef)

	return func(r chi.Router) {
		r.Use(lw.Middlewares(f.Logger)...)
		r.Use(middleware.RequestID, middleware.RealIP, c.Handler, CleanRequestPath, OutOfOrderMw(f))

		r.Method(http.MethodGet, "/", HandleItem(f))
		r.Method(http.MethodHead, "/", HandleItem(f))
		// TODO(marius): we can separate here the FedBOX specific collections from the ActivityPub spec ones
		//   using some regular expressions
		//   Eg: "/{collection:(inbox|outbox|followed)}"
		//   Eg: "/{collection:(activities|objects|actors|moderators|ignored|blocked|flagged)}"
		r.Route("/{collection}", f.CollectionRoutes(true))

		debugMw := func() http.Handler {
			if f.Conf.Env.IsDev() || f.debugMode.Load() {
				return middleware.Profiler()
			}
			return errors.NotFound
		}
		r.Mount("/debug", debugMw())

		r.Handle("/favicon.ico", errors.NotFound)
		r.NotFound(errors.NotFound.ServeHTTP)
		r.MethodNotAllowed(errors.HandleError(errors.MethodNotAllowedf("method not allowed")).ServeHTTP)
	}
}

func (f *FedBOX) CollectionRoutes(descend bool) func(chi.Router) {
	return func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Method(http.MethodGet, "/", HandleCollection(f))
			r.Method(http.MethodHead, "/", HandleCollection(f))

			r.Route("/{id}", func(r chi.Router) {
				r.Method(http.MethodGet, "/", HandleItem(f))
				r.Method(http.MethodHead, "/", HandleItem(f))
				if descend {
					r.Route("/{collection}", f.CollectionRoutes(false))
				}
			})

			// NOTE(marius): dump received requests to disk for debugging purposes
			if basePath, err := f.Conf.BaseStoragePath(); err == nil {
				r = r.With(processing.RequestToDiskMw(basePath, f.debugMode.Load))
			}
			r.Method(http.MethodPost, "/", HandleActivity(f))
		})
	}
}
