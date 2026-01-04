package fedbox

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	cache2 "git.sr.ht/~mariusor/cache"
	"git.sr.ht/~mariusor/lw"
	m "git.sr.ht/~mariusor/servermux"
	"git.sr.ht/~mariusor/storage-all"
	w "git.sr.ht/~mariusor/wrapper"
	"github.com/charmbracelet/ssh"
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

type Base struct {
	Conf              config.Options
	Logger            lw.Logger
	Service           vocab.Actor
	ServicePrivateKey []byte
	Storage           storage.FullStorage

	debugMode atomic.Bool
	sshServer *ssh.Server
}

type FedBOX struct {
	Base

	server m.Server
	R      chi.Router
	caches canStore

	maintenanceMode atomic.Bool
	shuttingDown    atomic.Bool

	keyGenerator func(act *vocab.Actor) error
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
		_ = os.RemoveAll(app.Conf.DefaultSocketPath())
		setters = append(setters, m.OnSocket(app.Conf.DefaultSocketPath()))
	}
	if app.Conf.Listen == "systemd" {
		lwCtx["systemd"] = true
		setters = append(setters, m.OnSystemd())
	} else if filepath.IsAbs(app.Conf.Listen) {
		dir := filepath.Dir(app.Conf.Listen)
		lwCtx["socket"] = app.Conf.Listen
		if _, err := os.Stat(dir); err == nil {
			setters = append(setters, m.OnSocket(app.Conf.Listen))
		}
	} else {
		lwCtx["tcp"] = app.Conf.Listen
		setters = append(setters, m.OnTCP(app.Conf.Listen))
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
		return nil, errors.Newf("invalid Storage")
	}
	if err := db.Open(); err != nil {
		return nil, errors.Annotatef(err, "unable to open Storage: %s", conf.StoragePath)
	}
	if conf.BaseURL == "" {
		return nil, errors.Newf("invalid empty BaseURL config")
	}

	app := FedBOX{
		Base: Base{
			Storage: db,
			Logger:  l,
			Conf:    conf,
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
		app.keyGenerator = AddKeyToPerson(metaSaver, keysType)
	}

	app.debugMode.Store(conf.Env.IsDev())

	if err := app.setupService(); err != nil {
		app.errFn("unable to save the instance's self service: %s", err)
		return nil, err
	}

	app.R.Group(app.Routes())

	muxSetters := []m.MuxFn{m.WriteWait(app.Conf.TimeOut)}
	if !app.Conf.Env.IsTest() && !app.Conf.Env.IsDev() {
		muxSetters = append(muxSetters, m.GracefulWait(defaultGraceWait))
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
	keysType := KeyTypeRSA
	if conf.MastodonIncompatible {
		keysType = KeyTypeED25519
	}
	if err != nil && errors.IsNotFound(err) {
		f.infFn("No service actor found, creating one: %s", selfIRI)
		self := ap.Self(selfIRI)
		if err = CreateService(db, self, keysType); err != nil {
			return err
		}
		f.Service = self
	}
	key, err := db.LoadKey(f.Service.ID)
	if err != nil {
		f.errFn("Unable to load the private key for the instance's Service: %s", err)
	}
	if key != nil {
		prvEnc, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			return err
		}
		r := pem.Block{Type: "PRIVATE KEY", Bytes: prvEnc}
		f.ServicePrivateKey = pem.EncodeToMemory(&r)
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
		_ = os.RemoveAll(f.Conf.DefaultSocketPath())
		if filepath.IsAbs(f.Conf.Listen) {
			if _, err := os.Stat(f.Conf.Listen); err == nil {
				_ = os.RemoveAll(f.Conf.Listen)
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
	logCtx := lw.Ctx{
		"URL": f.Conf.BaseURL,
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
	}).Exec(ctx, f.server.Start)
	if err == nil {
		logger.Infof("Stopped")
	}
	return err
}

func (b *Base) SendSignal(sig syscall.Signal) error {
	pid, err := b.Conf.ReadPid()
	if err != nil {
		return errors.Annotatef(err, "unable to read pid file")
	}
	return syscall.Kill(pid, sig)
}

func (b *Base) infFn(s string, p ...any) {
	if b.Logger != nil {
		b.Logger.Infof(s, p...)
	}
}

func (b *Base) errFn(s string, p ...any) {
	if b.Logger != nil {
		b.Logger.Errorf(s, p...)
	}
}

func CreateService(r storage.FullStorage, self vocab.Item, keyType string) (err error) {
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

	if err = AddKeyToItem(r, self, keyType); err != nil {
		return err
	}

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
			// NOTE(marius): create fedbox custom collections /activities, /objects, /actors
			if _, err := rr.Create(col(stream.GetID())); err != nil {
				multi = errors.Join(multi, err)
			}
		}
		return multi
	})
}
