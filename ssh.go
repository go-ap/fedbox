//go:build ssh

package fedbox

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"strings"

	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/mask"
	m "git.sr.ht/~mariusor/servermux"
	"github.com/alecthomas/kong"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	vocab "github.com/go-ap/activitypub"
	"github.com/muesli/termenv"
	"golang.org/x/crypto/ed25519"
)

func SSHAuthPw(f *FedBOX) ssh.PasswordHandler {
	return func(ctx ssh.Context, pw string) bool {
		acc, ok := pwCheck(f, ctx.User(), []byte(pw))
		if !ok {
			f.Logger.WithContext(lw.Ctx{"iri": ctx.User(), "pw": mask.S(pw)}).Warnf("failed password authentication")
			return false
		}

		ctx.SetValue("actor", acc)
		//if acc != nil && !f.Service.Equals(acc) {
		//	f.Logger.WithContext(lw.Ctx{"actor": acc.ID}).Warnf("actor is invalid for SSH access")
		//	return false
		//}
		return true
	}
}

func SSHAuthPublicKey(f *FedBOX) ssh.PublicKeyHandler {
	return func(ctx ssh.Context, key ssh.PublicKey) bool {
		acc, ok := publicKeyCheck(f, ctx.User(), key)
		if !ok {
			f.Logger.WithContext(lw.Ctx{"iri": ctx.User()}).Warnf("failed public key authentication")
			return false
		}

		ctx.SetValue("actor", acc)
		//if acc != nil && !f.Service.Equals(acc) {
		//	f.Logger.WithContext(lw.Ctx{"actor": acc.ID}).Warnf("actor is invalid for SSH access")
		//	return false
		//}
		return true
	}
}

func runSSHCommand(f *FedBOX, outIo, errIo io.Writer, args []string) {
	cmd := new(SSH)
	kongDefaultVars["name"] = "FedBOX SSH admin"
	kongDefaultVars["URL"] = string(f.Service.ID)
	k, err := kong.New(
		cmd,
		kong.Name(kongDefaultVars["name"]),
		kong.Description("${name} (version ${version}) ${URL}"),
		kongDefaultVars,
		kong.Writers(outIo, errIo),
	)
	if err != nil {
		_, _ = fmt.Fprintf(errIo, "Error: %s\n", err)
		return
	}
	ctx, err := k.Parse(args)
	if err != nil {
		_ = k.Errorf("%s\n", err)
		return
	}
	f.out = k.Stdout
	f.err = k.Stderr

	if err = ctx.Run(&f.Base); err != nil {
		_ = k.Errorf("%s\n", err)
		_ = ctx.PrintUsage(true)
	}
	return
}

func MainTui(f *FedBOX) wish.Middleware {
	teaHandler := func(s ssh.Session) *tea.Program {
		_, _, active := s.Pty()
		if !active {
			// NOTE(marius): this is not an interactive session, try to run the received command
			if len(s.Command()) == 0 {
				_, _ = fmt.Fprintln(s.Stderr(), "PTY is not interactive and no command was sent")
				_ = s.Exit(1)
				return nil
			}
			runSSHCommand(f, s, s.Stderr(), s.Command())
			_, _ = fmt.Fprintln(s)
			_ = s.Exit(0)
			return nil
		}
		_, _ = fmt.Fprintln(s.Stderr(), "Interactive PTYs are not supported")
		_ = s.Exit(1)

		return nil
	}

	return bm.MiddlewareWithProgramHandler(teaHandler, termenv.ANSI256)
}

func pwCheck(f *FedBOX, id string, pw []byte) (*vocab.Actor, bool) {
	maybeActor, err := f.Storage.Load(vocab.IRI(id))
	if err != nil {
		return nil, false
	}
	actor, err := vocab.ToActor(maybeActor)
	if err != nil {
		return nil, false
	}
	err = f.Storage.PasswordCheck(actor.ID, pw)
	if err != nil {
		return nil, false
	}
	return actor, true
}

func publicKeyCheck(f *FedBOX, id string, sessKey ssh.PublicKey) (*vocab.Actor, bool) {
	maybeActor, err := f.Storage.Load(vocab.IRI(id))
	if err != nil {
		return nil, false
	}
	var key crypto.PublicKey
	err = vocab.OnActor(maybeActor, func(actor *vocab.Actor) error {
		servicePubKey := actor.PublicKey.PublicKeyPem
		pubBytes, _ := pem.Decode([]byte(servicePubKey))
		key, err = x509.ParsePKIXPublicKey(pubBytes.Bytes)
		if err != nil {
			if key, err = x509.ParsePKCS1PublicKey(pubBytes.Bytes); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, false
	}

	switch pub := key.(type) {
	case *rsa.PublicKey:
		return &f.Service, !pub.Equal(sessKey)
	case *ecdsa.PublicKey:
		return &f.Service, !pub.Equal(sessKey)
	case ed25519.PublicKey:
		return &f.Service, !pub.Equal(sessKey)
	default:
		return nil, false
	}
}

func initSSHServer(app *FedBOX) (m.Server, error) {
	if app.Conf.SSHPort <= 0 {
		return nil, nil
	}
	listen := app.Conf.Listen
	if portIndex := strings.Index(listen, ":"); portIndex > 0 {
		listen = listen[:portIndex]
	}
	app.Logger.WithContext(lw.Ctx{"ssh": listen, "port": app.Conf.SSHPort}).Debugf("Accepting SSH requests")
	initFns := []m.SSHSetFn{
		wish.WithAddress(fmt.Sprintf("%s:%d", listen, app.Conf.SSHPort)),
		wish.WithPublicKeyAuth(SSHAuthPublicKey(app)),
		wish.WithPasswordAuth(SSHAuthPw(app)),
		wish.WithMiddleware(
			logging.MiddlewareWithLogger(justPrintLogger(app.Logger.Debugf)),
			//activeterm.Middleware(),
			MainTui(app),
		),
	}
	if app.ServicePrivateKey != nil {
		// NOTE(marius): use the service private key as a host key
		initFns = append(initFns, wish.WithHostKeyPEM(app.ServicePrivateKey))
	}
	return m.SSHServer(initFns...)
}
