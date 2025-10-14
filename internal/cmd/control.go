package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"os"
	"time"

	"git.sr.ht/~mariusor/lw"
	"github.com/alecthomas/kong"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	st "github.com/go-ap/fedbox/storage"
	"golang.org/x/crypto/ssh/terminal"
)

type Control struct {
	Conf    config.Options
	Logger  lw.Logger
	Service vocab.Actor
	Storage st.FullStorage
}

type Storage struct {
	Type config.StorageType `help:"Type of the backend to use. Possible values: ${storageTypes}"`

	Bootstrap      BootstrapCmd   `cmd:""`
	Reset          ResetCmd       `cmd:"" help:"Reset an existing storage."`
	FixCollections FixCollections `cmd:"" help:"Fix storage collections."`
}

type CTL struct {
	ctl *Control

	Url     url.URL          `help:"The URL used by the application."`
	Env     env.Type         `enum:"${envTypes}" help:"The environment to use. Expected values: ${envTypes}" default:"${defaultEnv}"`
	Verbose int              `counter:"v" help:"Increase verbosity level from the default associated with the environment settings."`
	Path    string           `path:"" help:"The path for the storage folder or socket" default:"." env:"STORAGE_PATH"`
	Version kong.VersionFlag `short:"V"`

	// Commands
	Pub         Pub         `cmd:"" name:"pub" alt:"ap" help:"ActivityPub management helper"`
	OAuth       OAuth       `cmd:"" name:"oauth"`
	Storage     Storage     `cmd:""`
	Accounts    Accounts    `cmd:"" help:"Accounts helper."`
	Maintenance Maintenance `cmd:"" help:"Toggle maintenance mode for the running FedBOX server."`
	Reload      Reload      `cmd:"" help:"Reload the running FedBOX server configuration."`
	Stop        Stop        `cmd:"" help:"Stops the running FedBOX server configuration."`
}

func InitControl(c *CTL, version string) (*Control, error) {
	opt := config.Options{
		Env:         c.Env,
		LogLevel:    lw.InfoLevel,
		AppName:     AppName,
		StoragePath: c.Path,
		Hostname:    c.Url.Host,
		Secure:      c.Url.Scheme == "https",
		BaseURL:     c.Url.String(),
		Version:     version,
	}
	if c.Verbose > 1 {
		opt.LogLevel = lw.DebugLevel
	}

	logger := lw.Prod(lw.SetOutput(os.Stderr), lw.SetLevel(opt.LogLevel))
	ct := Control{}
	if err := setup(&ct, opt, logger); err != nil {
		return nil, err
	}
	return &ct, nil
}

func New(db st.FullStorage, conf config.Options, l lw.Logger) (*Control, error) {
	self, err := ap.LoadActor(db, ap.DefaultServiceIRI(conf.BaseURL))
	if err != nil {
		l.Warnf("unable to load actor: %s", err)
	}

	return &Control{
		Conf:    conf,
		Service: self,
		Storage: db,
		Logger:  l,
	}, nil
}

func setup(ct *Control, options config.Options, l lw.Logger) error {
	environ := options.Env
	path := options.StoragePath
	typ := options.Storage
	conf, err := config.Load(path, environ, time.Second)
	if err != nil {
		l.Errorf("Unable to load config files for environment %s: %s", environ, err)
	}
	if typ != "" {
		conf.Storage = typ
	}
	if conf.StoragePath == "" && path != "." {
		conf.StoragePath = path
	}
	db, err := fedbox.Storage(conf, l)
	if err != nil {
		return err
	}
	if err = db.Open(); err != nil {
		return errors.Annotatef(err, "Unable to open FedBOX storage for path %s", conf.StoragePath)
	}
	defer db.Close()

	self, err := ap.LoadActor(db, ap.DefaultServiceIRI(conf.BaseURL))
	if err != nil {
		l.Warnf("unable to load actor: %s", err)
	}

	ct.Conf = conf
	ct.Service = self
	ct.Storage = db
	ct.Logger = l
	return nil
}

func loadPwFromStdin(confirm bool, s string, params ...any) ([]byte, error) {
	fmt.Printf(s+" pw: ", params...)
	pw1, _ := terminal.ReadPassword(0)
	fmt.Println()
	if confirm {
		fmt.Printf("pw again: ")
		pw2, _ := terminal.ReadPassword(0)
		fmt.Println()
		if !bytes.Equal(pw1, pw2) {
			return nil, errors.Errorf("Passwords do not match")
		}
	}
	return pw1, nil
}

func loadFromStdin(s string, params ...any) ([]byte, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(s+": ", params...)
	input, _ := reader.ReadBytes('\n')
	return input[:len(input)-1], nil
}

func Errf(s string, par ...any) {
	_, _ = fmt.Fprintf(os.Stderr, s+"\n", par...)
}
