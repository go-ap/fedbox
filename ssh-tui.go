//go:build ssh && tui

package fedbox

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
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
	return tea.NewProgram(motley.Model(f.Logger, st), tea.WithInput(s), tea.WithOutput(s))
}
