package containers

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/exec"
)

type SSHCmd struct {
	Cmd  []string
	User string
	Pw   []byte
	Key  crypto.PrivateKey
}

func (c SSHCmd) AsCommand() []string {
	return c.Cmd
}

func (c SSHCmd) Options() []exec.ProcessOption {
	prvEnc, err := x509.MarshalPKCS8PrivateKey(c.Key)
	if err != nil {
		return nil
	}

	// NOTE(marius): we pass the private key and the password through env variables.
	env := []string{
		"_PK=" + string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: prvEnc})),
		"_PW=" + string(c.Pw),
	}
	opts := []exec.ProcessOption{exec.WithEnv(env), exec.WithUser(c.User)}
	return opts
}

var _ tc.Executable = SSHCmd{}
