package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"time"

	"git.sr.ht/~mariusor/lw"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	st "github.com/go-ap/fedbox/storage"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh/terminal"
)

type Control struct {
	Conf    config.Options
	Logger  lw.Logger
	Service vocab.Actor
	Storage st.FullStorage
}

func New(db st.FullStorage, conf config.Options, l lw.Logger) *Control {
	self, _ := ap.LoadActor(db, ap.DefaultServiceIRI(conf.BaseURL))
	return &Control{
		Conf:    conf,
		Service: self,
		Storage: db,
		Logger:  l,
	}
}

var ctl Control

func Before(c *cli.Context) error {
	fields := lw.Ctx{}
	if c.Command != nil && c.Command.Name != "" {
		fields["cli"] = c.Command.FullName()
	}

	logger := lw.Dev(lw.SetOutput(os.Stderr))
	ct, err := setup(c, logger.WithContext(fields))
	if err != nil {
		// Ensure we don't print the default help message, which is not useful here
		c.App.CustomAppHelpTemplate = "Failed"
		logger.WithContext(lw.Ctx{"err": err}).Errorf("Error")
		return err
	}
	ctl = *ct

	return nil
}

func setup(c *cli.Context, l lw.Logger) (*Control, error) {
	environ := env.Type(c.String("env"))
	conf, err := config.LoadFromEnv(environ, time.Second)
	if err != nil {
		l.Errorf("Unable to load config files for environment %s: %s", environ, err)
	}
	path := c.String("path")
	if path != "." {
		conf.StoragePath = path
	}
	typ := c.String("type")
	if typ != "" {
		conf.Storage = config.StorageType(typ)
	}
	if conf.Storage == config.StoragePostgres {
		host := c.String("host")
		if host == "" {
			host = "localhost"
		}
		port := c.Int64("port")
		if port == 0 {
			host = path
		}
		user := c.String("user")
		if user == "" {
			user = "fedbox"
		}
		pw, err := loadPwFromStdin(true, "%s@%s's", user, host)
		if err != nil {
			return nil, err
		}
		_ = config.BackendConfig{
			Enabled: false,
			Host:    host,
			Port:    port,
			User:    user,
			Pw:      string(pw),
			Name:    user,
		}
	}
	db, err := fedbox.Storage(conf, l)
	if err != nil {
		return nil, err
	}
	return New(db, conf, l), nil
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
	fmt.Fprintf(os.Stderr, s+"\n", par...)
}
