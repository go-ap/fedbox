//go:build ssh

package fedbox

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/mask"
	"git.sr.ht/~mariusor/motley"
	m "git.sr.ht/~mariusor/servermux"
	"github.com/alecthomas/kong"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func runSSHCommand(f *FedBOX, s ssh.Session) error {
	args := s.Command()
	// NOTE(marius): this is not an interactive session, try to run the received command
	if len(args) == 0 {
		return fmt.Errorf("PTY is not interactive and no command was sent")
	}

	cmd := new(SSH)
	kongDefaultVars["name"] = "FedBOX SSH"
	kongDefaultVars["URL"] = string(f.Service.ID)
	k, err := kong.New(
		cmd,
		kong.UsageOnError(),
		kong.Name(kongDefaultVars["name"]),
		kong.Description("${name} (version ${version}) ${URL}"),
		kong.Writers(s, s.Stderr()),
		kong.Exit(func(_ int) {}),
		kongDefaultVars,
	)
	if err != nil {
		return err
	}
	ctx, err := k.Parse(args)
	if err != nil {
		return err
	}
	f.out = k.Stdout
	f.err = k.Stderr
	f.in = s

	if err = ctx.Run(&f.Base); err != nil {
		_ = k.Errorf("%s\n", err)
	}
	return err
}

func MainTui(f *FedBOX) wish.Middleware {
	teaHandler := func(s ssh.Session) *tea.Program {
		lwCtx := lw.Ctx{}
		acc, ok := s.Context().Value("actor").(*vocab.Actor)
		if ok {
			lwCtx["actor"] = acc.GetLink()
		}

		_, _, active := s.Pty()
		lwCtx["active"] = active

		f.Logger.WithContext(lwCtx).Infof("opening ssh session")
		// NOTE(marius): this is not an interactive session, try to run the received command
		if len(s.Command()) > 0 {
			if err := runSSHCommand(f, s); err != nil {
				_ = s.Exit(1)
			}
			_ = s.Exit(0)
			return nil
		}

		if !active {
			if len(s.Command()) == 0 {
				_, _ = fmt.Fprintln(s.Stderr(), "PTY is not interactive and no command was sent")
				_ = s.Exit(1)
			}
			return nil
		}

		// Set the global color profile to ANSI256 for Docker compatibility
		lipgloss.SetColorProfile(termenv.ANSI256)

		service, err := motley.FedBOX(f.Logger.WithContext(lwCtx), motley.Storage{
			FullStorage: f.Storage,
			Root:        acc,
		})
		if err != nil {
			_, _ = fmt.Fprintf(s.Stderr(), "Error: %s", err)
			_ = s.Exit(1)
		}
		return tea.NewProgram(motley.Model(service), tea.WithFPS(60), tea.WithInput(s), tea.WithOutput(s), tea.WithAltScreen())
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
		key, _ = x509.ParsePKIXPublicKey(pubBytes.Bytes)
		if key != nil {
			return nil
		}
		key, err = x509.ParsePKCS1PublicKey(pubBytes.Bytes)
		return err
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
	initFns := []m.SSHSetFn{
		wish.WithPublicKeyAuth(SSHAuthPublicKey(app)),
		wish.WithPasswordAuth(SSHAuthPw(app)),
		wish.WithMiddleware(
			logging.MiddlewareWithLogger(justPrintLogger(app.Logger.Debugf)),
			MainTui(app),
		),
	}

	listen := app.Conf.SSHListen()
	if len(listen) == 0 {
		app.Logger.Warnf("No valid SSH listen configurations")
		return nil, nil
	}
	initFns = append(initFns, wish.WithAddress(listen[0]))
	app.Logger.WithContext(lw.Ctx{"host": app.Conf.ListenHost, "port": app.Conf.SSHPort}).Debugf("Accepting SSH requests")
	if app.ServicePrivateKey != nil {
		// NOTE(marius): use the service private key as a host key
		if prvEnc, err := x509.MarshalPKCS8PrivateKey(app.ServicePrivateKey); err == nil {
			r := pem.Block{Type: "PRIVATE KEY", Bytes: prvEnc}
			initFns = append(initFns, wish.WithHostKeyPEM(pem.EncodeToMemory(&r)))
		}
	}
	return m.SSHServer(initFns...)
}
