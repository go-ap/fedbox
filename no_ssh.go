//go:build !ssh

package fedbox

import m "git.sr.ht/~mariusor/servermux"

func initSSHServer(app *FedBOX) (m.Server, error) {
	return nil, nil
}
