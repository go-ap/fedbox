package fedbox

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"git.sr.ht/~mariusor/lw"
	m "git.sr.ht/~mariusor/servermux"
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
)

func init() {
	// set local path typer to validate collections
	processing.Typer = pathTyper{}
}

type LogFn func(string, ...any)

type canStore = cache.CanStore

type FedBOX struct {
	Base

	server m.Server
	R      chi.Router
	caches canStore

	maintenanceMode atomic.Bool
	shuttingDown    atomic.Bool

	keyGenerator func(act *vocab.Actor) error
}

const defaultGraceWait = 1500 * time.Millisecond

func initHttpServer(app *FedBOX) (m.Server, error) {
	setters := []m.SetFn{m.Handler(app.R)}

	lwCtx := lw.Ctx{}
	if app.Conf.Secure {
		if len(app.Conf.CertPath)+len(app.Conf.KeyPath) > 0 {
			setters = append(setters, m.WithTLSCert(app.Conf.CertPath, app.Conf.KeyPath))
			lwCtx["TLS"] = true
		} else {
			app.Conf.Secure = false
		}
	}

	// NOTE(marius): we now set-up a default socket listener
	if !app.Conf.Env.IsTest() {
		_ = os.RemoveAll(app.Conf.InternalSocketPath())
		setters = append(setters, m.OnSocket(app.Conf.InternalSocketPath()))
	}

	listenHTTP := app.Conf.HTTPListen()
	if len(listenHTTP) == 0 {
		return nil, errors.Newf("No valid HTTP listen configurations")
	}
	for _, pathOrHost := range listenHTTP {
		if pathOrHost == "systemd" {
			lwCtx["systemd"] = true
			setters = append(setters, m.OnSystemd())
		} else if filepath.IsAbs(pathOrHost) {
			dir := filepath.Dir(pathOrHost)
			lwCtx["socket"] = pathOrHost
			if _, err := os.Stat(dir); err == nil {
				setters = append(setters, m.OnSocket(pathOrHost))
			}
		} else {
			lwCtx["host"] = app.Conf.ListenHost
			lwCtx["port"] = app.Conf.HTTPPort
			setters = append(setters, m.OnTCP(pathOrHost))
		}
	}

	httpSrv, err := m.HttpServer(setters...)
	if err != nil {
		return nil, err
	}
	app.Logger.WithContext(lwCtx).Debugf("Accepting HTTP requests")
	return httpSrv, nil
}

// New instantiates a new FedBOX instance
func New(l lw.Logger, conf config.Options, db storage.FullStorage) (*FedBOX, error) {
	if db == nil {
		return nil, errors.Newf("invalid storage")
	}
	if err := db.Open(); err != nil {
		return nil, errors.Annotatef(err, "unable to open storage: %s", conf.StoragePath)
	}

	app := FedBOX{
		Base: Base{
			Storage: db,
			Logger:  l,
			Conf:    conf,
			out:     os.Stdout,
			err:     os.Stderr,
			in:      os.Stdin,
		},
		R:      chi.NewRouter(),
		caches: cache.New(conf.RequestCache),
	}

	if metaSaver, ok := db.(storage.MetadataStorage); ok {
		keysType := "ED25519"
		if conf.MastodonIncompatible {
			keysType = "RSA"
		}

		l.Debugf("Setting actor key generator %T[%s]", metaSaver, keysType)
		app.keyGenerator = ap.AddKeyToPerson(metaSaver, keysType)
	}

	app.debugMode.Store(conf.Env.IsDev())

	if err := app.setupService(); err != nil {
		return nil, err
	}

	app.R.Group(app.Routes())

	muxSetters := make([]m.MuxFn, 0, 1)
	if !app.Conf.Env.IsTest() && !app.Conf.Env.IsDev() {
		muxSetters = append(muxSetters, m.GracefulWait(app.Conf.TimeOut))
	}

	// NOTE(marius): we initialize the HTTP Server
	httpSrv, err := initHttpServer(&app)
	if err != nil {
		return nil, err
	}
	muxSetters = append(muxSetters, m.WithServer(httpSrv))

	sshServ, err := initSSHServer(&app)
	if err != nil {
		app.Logger.WithContext(lw.Ctx{"err": err}).Errorf("unable to open SSH connection")
	}
	if sshServ != nil {
		// NOTE(marius): if the SSH Server could be initialized
		muxSetters = append(muxSetters, m.WithServer(sshServ))
	}

	app.server, err = m.Mux(muxSetters...)
	if err != nil {
		return nil, err
	}

	return &app, nil
}

func (f *FedBOX) setupService() error {
	db := f.Storage

	conf := f.Conf

	selfIRI := ap.DefaultServiceIRI(conf.BaseURL)
	var err error

	f.Service, err = ap.LoadActor(db, selfIRI)
	keysType := ap.KeyTypeRSA
	if conf.MastodonIncompatible {
		keysType = ap.KeyTypeED25519
	}
	if err != nil && errors.IsNotFound(err) {
		f.Logger.Tracef("No service actor found, creating one: %s", selfIRI)
		self := ap.Self(selfIRI)
		if err = ap.CreateService(db, self, keysType); err != nil {
			return err
		}
		f.Service = self
	}
	if key, _ := db.LoadKey(f.Service.ID); key != nil {
		f.ServicePrivateKey = key
	}

	return nil
}

func (f *FedBOX) Pause() error {
	if f.maintenanceMode.Load() {
		// restart everything
		f.Storage.Close()
	} else {
		return f.Storage.Open()
	}
	return nil
}

// Stop
func (f *FedBOX) Stop(ctx context.Context) error {
	f.Storage.Close()

	f.shuttingDown.Store(true)
	defer func() {
		_ = os.RemoveAll(f.Conf.PidPath())
		_ = os.RemoveAll(f.Conf.InternalSocketPath())
		if filepath.IsAbs(f.Conf.SocketPath) {
			if _, err := os.Stat(f.Conf.SocketPath); err == nil {
				_ = os.RemoveAll(f.Conf.SocketPath)
			}
		}
	}()

	return f.server.Stop(ctx)
}

func (f *FedBOX) reload() (err error) {
	f.Conf, err = config.Load(".", f.Conf.Env, f.Conf.TimeOut)
	f.caches.Delete()
	return err
}

func (f *FedBOX) actorFromRequestWithClient(r *http.Request, cl *client.C, receivedIn vocab.IRI) vocab.Actor {
	// NOTE(marius): if the Storage is nil, we can still use the remote client in the load function
	isLocalFn := func(iri vocab.IRI) bool {
		return iri.Contains(vocab.IRI(f.Conf.BaseURL), true)
	}

	var logFn auth.LoggerFn = func(ctx lw.Ctx, msg string, p ...interface{}) {
		f.Logger.WithContext(ctx).Debugf(msg, p...)
	}

	initFns := []auth.SolverInitFn{
		auth.SolverWithLogger(logFn),
		auth.SolverWithStorage(f.Storage),
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
		f.Logger.WithContext(lw.Ctx{"err": err.Error()}).Errorf("unable to load an authorized Actor from request")
	}
	return actor
}

// Run is the wrapper for starting the web-server and handling signals
func (f *FedBOX) Run(ctx context.Context) error {
	logCtx := lw.Ctx{}
	if f.Conf.BaseURL != "" {
		logCtx["URL"] = f.Conf.BaseURL
	}
	if f.Conf.Version != "" {
		logCtx["version"] = f.Conf.Version
	}
	var cancelFn func()

	ctx, cancelFn = context.WithCancel(ctx)
	defer cancelFn()

	logger := f.Logger.WithContext(logCtx)
	logger.Infof("Started")
	if err := f.Conf.WritePid(); err != nil {
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
			logger.WithContext(lw.Ctx{"wait": f.Conf.TimeOut}).Debugf("SIGINT received, interrupted")
			exitWithErrOrInterrupt(f.Stop(ctx), exit)
		},
		syscall.SIGTERM: func(exit chan<- error) {
			logger.WithContext(lw.Ctx{"wait": f.Conf.TimeOut}).Debugf("SIGTERM received, stopping with cleanup")
			exitWithErrOrInterrupt(f.Stop(ctx), exit)
		},
		syscall.SIGQUIT: func(exit chan<- error) {
			logger.Debugf("SIGQUIT received, ungraceful force stopping")
			// NOTE(marius): to skip any graceful wait on the listening server, cancel the context first
			cancelFn()
			exitWithErrOrInterrupt(f.Stop(ctx), exit)
		},
	}).Exec(ctx, f.server.Start)
	if err == nil {
		logger.Infof("Stopped")
	}
	return err
}
