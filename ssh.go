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
		return true
	}
}

func runSSHCommand(f *FedBOX, s ssh.Session) error {
	args := s.Command()
	// NOTE(marius): this is not an interactive session, try to run the received command
	if len(args) == 0 {
		return fmt.Errorf("PTY is not interactive and no command was sent")
	}
	ctl := new(Base)
	ctl.Conf = f.Conf
	ctl.Logger = f.Logger
	ctl.Service = f.Service
	ctl.ServicePrivateKey = f.ServicePrivateKey
	ctl.Storage = f.Storage
	ctl.out = s
	ctl.in = s
	ctl.err = s.Stderr()

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
		_ = k.Errorf("%s\n", err)
		return err
	}

	if err = ctx.Run(ctl); err != nil {
		_ = k.Errorf("%s\n", err)
		return err
	}
	_ = k.Printf("OK\n")
	return nil
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
	actorIRI := vocab.IRI(id)
	maybeActor, err := f.Storage.Load(actorIRI)
	if err != nil {
		if f.Service.ID.Equals(actorIRI, false) {
			maybeActor = f.Service
		} else {
			return nil, false
		}
	}
	var key crypto.PublicKey
	var actor *vocab.Actor
	err = vocab.OnActor(maybeActor, func(act *vocab.Actor) error {
		servicePubKey := act.PublicKey.PublicKeyPem
		actor = act
		if pubBytes, _ := pem.Decode([]byte(servicePubKey)); pubBytes != nil {
			key, _ = x509.ParsePKIXPublicKey(pubBytes.Bytes)
			if key != nil {
				return nil
			}
			key, err = x509.ParsePKCS1PublicKey(pubBytes.Bytes)
		}
		return err
	})
	if err != nil {
		return nil, false
	}

	switch pub := key.(type) {
	case *rsa.PublicKey:
		return actor, !pub.Equal(sessKey)
	case *ecdsa.PublicKey:
		return actor, !pub.Equal(sessKey)
	case ed25519.PublicKey:
		return actor, !pub.Equal(sessKey)
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
