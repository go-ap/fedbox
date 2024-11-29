package fedbox

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"syscall"

	"git.sr.ht/~mariusor/lw"
	w "git.sr.ht/~mariusor/wrapper"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/cache"
	"github.com/go-ap/client"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	st "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/processing"
	"github.com/go-chi/chi/v5"
	"github.com/openshift/osin"
)

func init() {
	// set local path typer to validate collections
	processing.Typer = pathTyper{}
}

type LogFn func(string, ...any)

type canStore = cache.CanStore

type FedBOX struct {
	R       chi.Router
	conf    config.Options
	self    vocab.Service
	client  client.C
	storage st.FullStorage
	version string
	caches  canStore
	logger  lw.Logger

	keyGenerator func(act *vocab.Actor) error

	startFn func(ctx context.Context) error
	stopFn  func(ctx context.Context) error
}

var emptyCtxtFn = func(_ context.Context) error {
	return nil
}

var InternalIRI = vocab.IRI("https://fedbox/")

// New instantiates a new FedBOX instance
func New(l lw.Logger, ver string, conf config.Options, db st.FullStorage) (*FedBOX, error) {
	if db == nil {
		return nil, errors.Newf("invalid storage")
	}
	if conf.BaseURL == "" {
		return nil, errors.Newf("invalid empty BaseURL config")
	}
	app := FedBOX{
		version: ver,
		conf:    conf,
		R:       chi.NewRouter(),
		storage: db,
		logger:  l,
		caches:  cache.New(conf.RequestCache),

		startFn: emptyCtxtFn,
		stopFn:  emptyCtxtFn,
	}

	if metaSaver, ok := db.(st.MetadataTyper); ok {
		keysType := "ED25519"
		if conf.MastodonCompatible {
			keysType = "RSA"
		}

		l.Infof("Setting actor key generator %T[%s]", metaSaver, keysType)
		app.keyGenerator = AddKeyToPerson(metaSaver, keysType)
	}

	errors.IncludeBacktrace = conf.LogLevel == lw.TraceLevel

	selfIRI := ap.DefaultServiceIRI(conf.BaseURL)
	app.self, _ = ap.LoadActor(db, selfIRI)
	if app.self.ID != selfIRI {
		if saver, ok := db.(st.CanBootstrap); ok {
			app.infFn("trying to bootstrap the instance's self service")
			app.self = ap.Self(selfIRI)
			if err := saver.CreateService(app.self); err != nil {
				app.errFn("unable to save the instance's self service: %s", err)
				return nil, err
			}
		}
		keysType := KeyTypeED25519
		if conf.MastodonCompatible {
			keysType = KeyTypeRSA
		}
		if saver, ok := db.(st.MetadataTyper); ok {
			if err := AddKeyToPerson(saver, keysType)(&app.self); err != nil {
				app.errFn("unable to save the instance's self service public key: %s", err)
			}
		}
	}

	app.client = *client.New(
		client.WithLogger(l.WithContext(lw.Ctx{"log": "client"})),
		client.SkipTLSValidation(!conf.Env.IsProd()),
	)

	app.R.Group(app.Routes())

	sockType := ""
	setters := []w.SetFn{w.Handler(app.R)}

	if app.conf.Secure {
		if len(app.conf.CertPath)+len(app.conf.KeyPath) > 0 {
			setters = append(setters, w.WithTLSCert(app.conf.CertPath, app.conf.KeyPath))
		} else {
			app.conf.Secure = false
		}
	}

	if app.conf.Listen == "systemd" {
		sockType = "Systemd"
		setters = append(setters, w.OnSystemd())
	} else if filepath.IsAbs(app.conf.Listen) {
		dir := filepath.Dir(app.conf.Listen)
		if _, err := os.Stat(dir); err == nil {
			sockType = "socket"
			setters = append(setters, w.OnSocket(app.conf.Listen))
			defer func() {
				if err := os.RemoveAll(app.conf.Listen); err != nil {
					app.logger.Errorf("Failed cleaning up: %s", err)
				}
			}()
		}
	} else {
		sockType = "TCP"
		setters = append(setters, w.OnTCP(app.conf.Listen))
	}

	// Get start/stop functions for the http server
	app.startFn, app.stopFn = w.HttpServer(setters...)
	app.conf.Listen += "[" + sockType + "]"

	return &app, nil
}

func (f *FedBOX) Config() config.Options {
	return f.conf
}

func (f *FedBOX) Storage() st.FullStorage {
	return f.storage
}

// Stop
func (f *FedBOX) Stop(ctx context.Context) {
	if r, ok := f.storage.(osin.Storage); ok {
		r.Close()
	}

	_ = f.stopFn(ctx)
}

func (f *FedBOX) reload() (err error) {
	f.conf, err = config.LoadFromEnv(f.conf.Env, f.conf.TimeOut)
	f.caches.Delete()
	return err
}

func (f *FedBOX) actorFromRequest(r *http.Request) vocab.Actor {
	// NOTE(marius): if the storage is nil, we can still use the remote client in the load function
	isLocalFn := func(iri vocab.IRI) bool {
		return iri.Contains(vocab.IRI(f.conf.BaseURL), true)
	}
	ar := auth.ClientResolver(&f.client, auth.SolverWithLogger(f.logger),
		auth.SolverWithStorage(f.storage), auth.SolverWithLocalIRIFn(isLocalFn))
	act, err := ar.LoadActorFromRequest(r)

	if err != nil {
		f.logger.Errorf("unable to load an authorized Actor from request: %+s", err)
	}
	return act
}

// Run is the wrapper for starting the web-server and handling signals
func (f *FedBOX) Run(ctx context.Context) error {
	logCtx := lw.Ctx{
		"URL":      f.conf.BaseURL,
		"version":  f.version,
		"listenOn": f.conf.Listen,
		"TLS":      f.conf.Secure,
	}

	logger := f.logger.WithContext(logCtx)

	logger.Infof("Starting")
	err := w.RegisterSignalHandlers(w.SignalHandlers{
		syscall.SIGHUP: func(_ chan<- error) {
			logger.Infof("SIGHUP received, reloading configuration")
			if err := f.reload(); err != nil {
				logger.Errorf("Failed: %+s", err.Error())
			}
		},
		syscall.SIGINT: func(exit chan<- error) {
			logger.Infof("SIGINT received, stopping")
			exit <- nil
		},
		syscall.SIGTERM: func(exit chan<- error) {
			logger.Infof("SIGITERM received, force stopping")
			exit <- nil
		},
		syscall.SIGQUIT: func(exit chan<- error) {
			logger.Infof("SIGQUIT received, force stopping with core-dump")
			f.Stop(ctx)
			exit <- nil
		},
	}).Exec(ctx, f.startFn)
	if err == nil {
		logger.Infof("Shutting down")
	}
	return err
}

func (f *FedBOX) infFn(s string, p ...any) {
	if f.logger != nil {
		f.logger.Infof(s, p...)
	}
}

func (f *FedBOX) errFn(s string, p ...any) {
	if f.logger != nil {
		f.logger.Errorf(s, p...)
	}
}
