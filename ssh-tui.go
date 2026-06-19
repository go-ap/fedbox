//go:build ssh && tui

package fedbox

import (
	"fmt"
	"log/slog"

	tea "charm.land/bubbletea/v2"
	"charm.land/wish/v2/bubbletea"
	"git.sr.ht/~mariusor/motley"
	"github.com/charmbracelet/ssh"
	vocab "github.com/go-ap/activitypub"
)

func wishTUI(f *FedBOX, s ssh.Session) *tea.Program {
	env := string(f.Conf.Env)
	acc, ok := s.Context().Value("actor").(*vocab.Actor)
	if !ok {
		_, _ = fmt.Fprintln(s.Stderr(), "Invalid actor for interactive session")
		_ = s.Exit(1)
	}

	st := motley.WithStore(f.Storage, acc, env)
	initFns := []tea.ProgramOption{tea.WithoutSignalHandler()}
	initFns = append(initFns, bubbletea.MakeOptions(s)...)
	return tea.NewProgram(motley.Model(slog.New(slog.DiscardHandler), st), initFns...)
}
