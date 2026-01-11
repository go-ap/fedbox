package fedbox

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"

	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/storage-all"
	"github.com/alecthomas/kong"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"golang.org/x/crypto/ssh/terminal"
)

type Storage struct {
	Type storage.Type `help:"Type of the backend to use. Possible values: ${storageTypes}"`

	Bootstrap      BootstrapCmd   `cmd:"" help:"Bootstrap the storage"`
	Reset          ResetCmd       `cmd:"" help:"Reset an existing storage."`
	FixCollections FixCollections `cmd:"" help:"Fix storage collections."`
}

type SSH struct {
	Pub         Pub         `cmd:"" name:"pub" alt:"ap" help:"ActivityPub management helper"`
	OAuth       OAuth       `cmd:"" name:"oauth"`
	Storage     Storage     `cmd:""`
	Accounts    Accounts    `cmd:"" help:"Accounts helper."`
	Maintenance Maintenance `cmd:"" help:"Toggle maintenance mode for the running FedBOX server."`
	Reload      Reload      `cmd:"" help:"Reload the running FedBOX server configuration."`
	Stop        Stop        `cmd:"" help:"Stops the running FedBOX server configuration."`
}

type CTL struct {
	SSH `embed:""`

	Url     url.URL          `help:"The URL used by the application."`
	Env     env.Type         `enum:"${envTypes}" help:"The environment to use. Expected values: ${envTypes}" default:"${defaultEnv}"`
	Verbose int              `counter:"v" help:"Increase verbosity level from the default associated with the environment settings."`
	Path    string           `path:"" help:"The path for the storage folder or socket" default:"." env:"STORAGE_PATH"`
	Version kong.VersionFlag `short:"V"`

	// Commands
	Run Serve `cmd:"" name:"run" help:"Run the ${name} instance server (version: ${version})" default:"withargs"`
}

func InitControl(c *CTL, version string) (*Base, error) {
	opt := config.Options{
		Env:         c.Env,
		LogLevel:    lw.InfoLevel,
		AppName:     AppName,
		StoragePath: c.Path,
		Hostname:    c.Url.Host,
		Secure:      c.Url.Scheme == "https",
		BaseURL:     c.Url.String(),
		Version:     version,
		TimeOut:     c.Run.Wait,
	}
	if c.Verbose > 1 {
		opt.LogLevel = lw.DebugLevel
	}

	errors.SetIncludeBacktrace(opt.LogLevel == lw.TraceLevel)
	ct := Base{
		in:  os.Stdin,
		out: os.Stdout,
		err: os.Stderr,
	}
	if err := setup(&ct, opt); err != nil {
		return nil, err
	}
	return &ct, nil
}

func NewBase(db storage.FullStorage, conf config.Options, l lw.Logger) (*Base, error) {
	return &Base{
		Conf:    conf,
		Storage: db,
		Logger:  l,
		in:      os.Stdin,
		out:     os.Stdout,
		err:     os.Stderr,
	}, nil
}

func setup(ct *Base, options config.Options) error {
	environ := options.Env
	path := options.StoragePath
	conf, err := config.Load(path, environ)
	if err != nil {
		return err
	}

	var out io.WriteCloser
	if conf.LogOutput != "" {
		if out, err = os.Open(conf.LogOutput); err != nil {
			return errors.Newf("Unable to output logs to %s: %s", conf.LogOutput, err)
		}
		defer func() {
			if err := out.Close(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Unable to close log output: %s", err)
			}
		}()
	}
	var l lw.Logger
	if conf.Env.IsDev() {
		l = lw.Dev(lw.SetLevel(conf.LogLevel), lw.SetOutput(out))
	} else {
		l = lw.Prod(lw.SetLevel(conf.LogLevel), lw.SetOutput(out))
	}

	typ := options.Storage
	if typ != "" {
		conf.Storage = typ
	}
	if conf.StoragePath == "" && path != "." {
		conf.StoragePath = path
	}
	db, err := storage.New(conf.StorageInitFns(l)...)
	if err != nil {
		return err
	}
	if err = db.Open(); err != nil {
		return errors.Annotatef(err, "Unable to open FedBOX storage for path %s", conf.StoragePath)
	}
	defer db.Close()

	ct.Conf = conf
	ct.Storage = db
	ct.Logger = l
	return nil
}

type muxReadWriter struct {
	io.Reader
	io.Writer
}

func (w muxReadWriter) Read(p []byte) (n int, err error) {
	if w.Reader != nil {
		return w.Reader.Read(p)
	}
	return 0, nil
}

func (w muxReadWriter) Write(p []byte) (n int, err error) {
	if w.Writer != nil {
		return w.Writer.Write(p)
	}
	return 0, nil
}

var _ io.Reader = muxReadWriter{}

func loadPwFromStdin(rw io.ReadWriter, prompt string) ([]byte, error) {
	term := terminal.NewTerminal(rw, "")
	pw1, _ := term.ReadPassword(prompt)
	p2 := "password again: "
	pw2, _ := term.ReadPassword(p2)
	if pw1 != pw2 {
		return nil, errors.Errorf("Passwords do not match")
	}
	return []byte(pw1), nil
}

func loadFromStdin(s string, params ...any) ([]byte, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(s+": ", params...)
	input, _ := reader.ReadBytes('\n')
	return input[:len(input)-1], nil
}

func Errf(out io.Writer, s string, par ...any) {
	_, _ = fmt.Fprintf(out, s+"\n", par...)
}
