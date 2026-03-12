//go:build ssh && !tui

package fedbox

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/ssh"
)

func wishTUI(_ *FedBOX, s ssh.Session) *tea.Program {
	_, _ = fmt.Fprintln(s.Stderr(), "Server does not support interactive sessions.")
	_ = s.Exit(1)
	return nil
}
