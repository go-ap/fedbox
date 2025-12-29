package fedbox

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	cache2 "git.sr.ht/~mariusor/cache"
	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/storage-all"
	w "git.sr.ht/~mariusor/wrapper"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/cache"
	"github.com/go-ap/client"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
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
	storage storage.FullStorage
	caches  canStore
	logger  lw.Logger

	debugMode       atomic.Bool
	maintenanceMode atomic.Bool
	shuttingDown    atomic.Bool

	keyGenerator func(act *vocab.Actor) error

	startFn func(ctx context.Context) error
	stopFn  func(ctx context.Context) error
}

var emptyCtxtFn = func(_ context.Context) error {
	return nil
}

var InternalIRI = vocab.IRI("https://fedbox/")

func Client(tr http.RoundTripper, conf config.Options, l lw.Logger) *client.C {
	cachePath, err := os.UserCacheDir()
	if err != nil {
		cachePath = os.TempDir()
	}

	if tr == nil {
		tr = &http.Transport{}
	}

	ua := fmt.Sprintf("%s@%s (+%s)", conf.BaseURL, conf.Version, ap.ProjectURL)
	baseClient := &http.Client{
		Transport: client.UserAgentTransport(ua, cache2.Private(tr, cache2.FS(filepath.Join(cachePath, conf.AppName)))),
	}

	return client.New(
		client.WithLogger(l.WithContext(lw.Ctx{"log": "client"})),
		client.WithHTTPClient(baseClient),
		client.SkipTLSValidation(!conf.Env.IsProd()),
	)
}

const defaultGraceWait = 1500 * time.Millisecond

// New instantiates a new FedBOX instance
func New(l lw.Logger, conf config.Options, db storage.FullStorage) (*FedBOX, error) {
	if db == nil {
		return nil, errors.Newf("invalid storage")
	}
	if err := db.Open(); err != nil {
		return nil, errors.Annotatef(err, "unable to open storage: %s", conf.StoragePath)
	}
	if conf.BaseURL == "" {
		return nil, errors.Newf("invalid empty BaseURL config")
	}

	app := FedBOX{
		conf:    conf,
		R:       chi.NewRouter(),
		storage: db,
		logger:  l,
		caches:  cache.New(conf.RequestCache),

		startFn: emptyCtxtFn,
		stopFn:  emptyCtxtFn,
	}

	if metaSaver, ok := db.(storage.MetadataStorage); ok {
		keysType := "ED25519"
		if conf.MastodonIncompatible {
			keysType = "RSA"
		}

		l.Debugf("Setting actor key generator %T[%s]", metaSaver, keysType)
		app.keyGenerator = AddKeyToPerson(metaSaver, keysType)
	}

	errors.SetIncludeBacktrace(conf.LogLevel == lw.TraceLevel)
	app.debugMode.Store(conf.Env.IsDev())

	if err := app.setupService(); err != nil {
		app.errFn("unable to save the instance's self service: %s", err)
		return nil, err
	}

	app.R.Group(app.Routes())

	sockType := ""
	setters := []w.SetFn{w.Handler(app.R), w.WriteWait(app.conf.TimeOut)}

	if app.conf.Secure {
		if len(app.conf.CertPath)+len(app.conf.KeyPath) > 0 {
			setters = append(setters, w.WithTLSCert(app.conf.CertPath, app.conf.KeyPath))
		} else {
			app.conf.Secure = false
		}
	}

	// NOTE(marius): we now set-up a default socket listener
	if !app.conf.Env.IsTest() {
		_ = os.RemoveAll(app.conf.DefaultSocketPath())
		setters = append(setters, w.OnSocket(app.conf.DefaultSocketPath()), w.GracefulWait(defaultGraceWait))
	}
	if app.conf.Listen == "systemd" {
		sockType = "Systemd"
		setters = append(setters, w.OnSystemd())
	} else if filepath.IsAbs(app.conf.Listen) {
		dir := filepath.Dir(app.conf.Listen)
		if _, err := os.Stat(dir); err == nil {
			sockType = "socket"
			setters = append(setters, w.OnSocket(app.conf.Listen))
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

func (f *FedBOX) setupService() error {
	db := f.storage

	conf := f.conf

	selfIRI := ap.DefaultServiceIRI(conf.BaseURL)
	var err error

	f.self, err = ap.LoadActor(db, selfIRI)
	if err != nil && errors.IsNotFound(err) {
		f.infFn("No service actor found, creating one: %s", selfIRI)
		self := ap.Self(selfIRI)
		if err = CreateService(db, self); err != nil {
			return err
		}
		f.self = self
		keysType := KeyTypeRSA
		if conf.MastodonIncompatible {
			keysType = KeyTypeED25519
		}
		if err = AddKeyToItem(db, &f.self, keysType); err != nil {
			f.errFn("Unable to save the instance's self service public key: %s", err)
		}
	}
	return nil
}

func (f *FedBOX) Config() config.Options {
	return f.conf
}

func (f *FedBOX) Storage() storage.FullStorage {
	return f.storage
}

func (f *FedBOX) Pause() error {
	if f.maintenanceMode.Load() {
		// restart everything
		f.storage.Close()
	} else {
		return f.storage.Open()
	}
	return nil
}

// Stop
func (f *FedBOX) Stop(ctx context.Context) error {
	f.storage.Close()

	f.shuttingDown.Store(true)
	defer func() {
		_ = os.RemoveAll(f.conf.PidPath())
		_ = os.RemoveAll(f.conf.DefaultSocketPath())
		if filepath.IsAbs(f.conf.Listen) {
			if _, err := os.Stat(f.conf.Listen); err == nil {
				_ = os.RemoveAll(f.conf.Listen)
			}
		}
	}()

	return f.stopFn(ctx)
}

func (f *FedBOX) reload() (err error) {
	f.conf, err = config.Load(".", f.conf.Env, f.conf.TimeOut)
	f.caches.Delete()
	return err
}

func (f *FedBOX) actorFromRequestWithClient(r *http.Request, cl *client.C, receivedIn vocab.IRI) vocab.Actor {
	// NOTE(marius): if the storage is nil, we can still use the remote client in the load function
	isLocalFn := func(iri vocab.IRI) bool {
		return iri.Contains(vocab.IRI(f.conf.BaseURL), true)
	}

	var logFn auth.LoggerFn = func(ctx lw.Ctx, msg string, p ...interface{}) {
		f.logger.WithContext(ctx).Debugf(msg, p...)
	}

	initFns := []auth.SolverInitFn{
		auth.SolverWithLogger(logFn),
		auth.SolverWithStorage(f.storage),
		auth.SolverWithLocalIRIFn(isLocalFn),
	}

	var ar auth.ActorVerifier
	switch {
	case processing.IsInbox(receivedIn):
		ar = auth.HTTPSignatureResolver(cl, initFns...)
	case processing.IsOutbox(receivedIn):
		ar = auth.OAuth2Resolver(cl, initFns...)
	default:
		ar = auth.Resolver(cl, initFns...)
	}

	actor, err := ar.Verify(r)
	if err != nil {
		f.logger.WithContext(lw.Ctx{"err": err.Error()}).Errorf("unable to load an authorized Actor from request")
	}
	return actor
}

// Run is the wrapper for starting the web-server and handling signals
func (f *FedBOX) Run(ctx context.Context) error {
	logCtx := lw.Ctx{
		"URL": f.conf.BaseURL,
	}
	if f.conf.Version != "" {
		logCtx["version"] = f.conf.Version
	}
	var cancelFn func()

	ctx, cancelFn = context.WithCancel(ctx)
	defer cancelFn()

	logger := f.logger.WithContext(logCtx)
	logger.WithContext(lw.Ctx{"listenOn": f.conf.Listen, "TLS": f.conf.Secure}).Infof("Started")
	if err := f.conf.WritePid(); err != nil {
		logger.Warnf("Unable to write pid file: %s", err)
		logger.Warnf("Some CLI commands relying on it will not work")
	}

	exitWithErrOrInterrupt := func(err error, exit chan<- error) {
		if err == nil {
			err = w.Interrupt
		}
		exit <- err
	}

	err := w.RegisterSignalHandlers(w.SignalHandlers{
		syscall.SIGHUP: func(_ chan<- error) {
			logger.Debugf("SIGHUP received, reloading configuration")
			if err := f.reload(); err != nil {
				logger.Errorf("Failed: %+s", err.Error())
			}
		},
		syscall.SIGUSR2: func(_ chan<- error) {
			isDebug := f.debugMode.Load()
			f.debugMode.Store(!isDebug)
			logger.WithContext(lw.Ctx{"debug": !isDebug}).Debugf("SIGUSR2 received, toggle debug mode")
		},
		syscall.SIGUSR1: func(_ chan<- error) {
			isMaintenance := f.maintenanceMode.Load()
			f.maintenanceMode.Store(!isMaintenance)

			logFn := logger.WithContext(lw.Ctx{"maintenance": !isMaintenance}).Debugf
			if err := f.Pause(); err != nil {
				logFn = logger.WithContext(lw.Ctx{"err": err.Error()}).Warnf
			}
			logFn("SIGUSR1 received, toggle maintenance mode")
		},
		syscall.SIGINT: func(exit chan<- error) {
			logger.WithContext(lw.Ctx{"wait": defaultGraceWait}).Debugf("SIGINT received, interrupted")
			exitWithErrOrInterrupt(f.Stop(ctx), exit)
		},
		syscall.SIGTERM: func(exit chan<- error) {
			logger.WithContext(lw.Ctx{"wait": defaultGraceWait}).Debugf("SIGTERM received, stopping with cleanup")
			exitWithErrOrInterrupt(f.Stop(ctx), exit)
		},
		syscall.SIGQUIT: func(exit chan<- error) {
			logger.Debugf("SIGQUIT received, ungraceful force stopping")
			// NOTE(marius): to skip any graceful wait on the listening server, cancel the context first
			cancelFn()
			exitWithErrOrInterrupt(f.Stop(ctx), exit)
		},
	}).Exec(ctx, f.startFn)
	if err == nil {
		logger.Infof("Stopped")
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

func CreateService(r storage.FullStorage, self vocab.Item) (err error) {
	_ = vocab.OnActor(self, func(service *vocab.Actor) error {
		service.Published = time.Now().UTC()
		return nil
	})
	self, err = r.Save(self)
	if err != nil {
		return err
	}

	c := osin.DefaultClient{Id: string(self.GetLink())}
	_ = r.CreateClient(&c)

	rr, ok := r.(storage.CollectionStore)
	if !ok {
		return nil
	}

	col := func(iri vocab.IRI) vocab.CollectionInterface {
		return &vocab.OrderedCollection{
			ID:           iri,
			Type:         vocab.OrderedCollectionType,
			Published:    time.Now().UTC(),
			AttributedTo: self.GetLink(),
			CC:           vocab.ItemCollection{vocab.PublicNS},
		}
	}
	return vocab.OnActor(self, func(service *vocab.Actor) error {
		var multi error
		for _, stream := range service.Streams {
			if _, err := rr.Create(col(stream.GetID())); err != nil {
				multi = errors.Join(multi, err)
			}
		}
		return multi
	})
}
