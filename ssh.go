//go:build ssh

package fedbox

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/mask"
	m "git.sr.ht/~mariusor/servermux"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	vocab "github.com/go-ap/activitypub"
	"github.com/muesli/termenv"
	"golang.org/x/crypto/ed25519"
)

// ResolveFilePathWithSubdir resolves a file path in a subdirectory
// Priority:
// 1. Local working directory (e.g., ./.ssh/stegodonhostkey)
// 2. User config directory (e.g., ~/.config/stegodon/.ssh/stegodonhostkey)
// 3. Returns the user config directory path if neither exists (for creation)
func ResolveFilePathWithSubdir(subdir, filename string) string {
	localPath := filepath.Join(subdir, filename)

	// Check local directory first
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}

	// Try user config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to local directory if we can't get config dir
		return localPath
	}

	userSubdir := filepath.Join(configDir, subdir)
	userPath := filepath.Join(userSubdir, filename)

	// If file exists in user dir, return that path
	if _, err := os.Stat(userPath); err == nil {
		return userPath
	}

	// Neither exists, create subdirectory and return user config path
	_ = os.MkdirAll(userSubdir, 0755)
	return userPath
}

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

func MainTui(f *FedBOX) wish.Middleware {
	teaHandler := func(s ssh.Session) *tea.Program {
		pty, _, active := s.Pty()
		if !active {
			wish.Println(s, "no active terminal, skipping")
			return nil
		}

		// Set the global color profile to ANSI256 for Docker compatibility
		lipgloss.SetColorProfile(termenv.ANSI256)
		lwCtx := lw.Ctx{"w": pty.Window.Width, "h": pty.Window.Height}

		acc, ok := s.Context().Value("actor").(*vocab.Actor)
		if ok {
			lwCtx["actor"] = acc.GetLink()
		}
		f.Logger.WithContext(lwCtx).Infof("opening ssh session")
		return nil //tea.NewProgram(nil, tea.WithFPS(60), tea.WithInput(s), tea.WithOutput(s), tea.WithAltScreen())
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
	return m.SSHServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", listen, app.Conf.SSHPort)),
		wish.WithHostKeyPEM(app.ServicePrivateKey),
		wish.WithPublicKeyAuth(SSHAuthPublicKey(app)),
		wish.WithPasswordAuth(SSHAuthPw(app)),
		wish.WithMiddleware(
			logging.MiddlewareWithLogger(corsLogger(app.Logger.Debugf)),
			activeterm.Middleware(),
			MainTui(app),
		),
	)
}
