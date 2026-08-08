package containers

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"strings"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client/c2s"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/exec"
)

type SSHCmd struct {
	Cmd  []string
	User string
	Pw   []byte
	IO   io.ReadWriter
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
	if c.IO != nil {
		opts = append(opts, WithIO(c.IO))
	}
	return opts
}

var _ tc.Executable = SSHCmd{}

type oauthextractorrw struct {
	Tok *c2s.BearerSigner
}

func (o *oauthextractorrw) Read(i []byte) (int, error) {
	return 0, nil
}

func (o *oauthextractorrw) Write(p []byte) (int, error) {
	return ExtractToken(o.Tok, p)
}

func ExtractToken(token *c2s.BearerSigner, i []byte) (int, error) {
	ll := len(i)
	if len(token.TokenType)+len(token.AccessToken) > 0 {
		// NOTE(marius): ignore all output after we received the bearer token
		return ll, nil
	}
	i = bytes.TrimSpace(i)
	auth, found := bytes.CutPrefix(i, []byte("Authorization: "))
	if !found {
		return ll, fmt.Errorf("unable to get Authorization value from CLI output: %s", i)
	}
	if hasEoL := bytes.IndexByte(auth, '\n'); hasEoL > 0 {
		auth = auth[:hasEoL]
	}
	authPieces := strings.Split(string(auth), " ")
	if len(authPieces) < 2 {
		return ll, fmt.Errorf("authorization value is not recognized: %+v", authPieces)
	}
	token.TokenType = strings.TrimSpace(authPieces[0])
	token.AccessToken = strings.TrimSpace(authPieces[1])
	if token.AccessToken == "" || token.TokenType == "" {
		return ll, fmt.Errorf("unable to build Authorization token")
	}
	return ll, nil
}

var _ io.ReadWriter = new(oauthextractorrw)

func ExecAs(user vocab.IRI, prv crypto.PrivateKey) admin {
	return admin{user: user, prv: prv}
}

type admin struct {
	user vocab.IRI
	prv  crypto.PrivateKey
}

type pwsetter struct {
	cnt int
	pw  string
}

var _ io.ReadWriter = new(pwsetter)

func (w *pwsetter) Read(i []byte) (int, error) {
	if w.cnt < 2 {
		w.cnt++
		pw := append([]byte(w.pw), '\n')
		l := copy(i, pw)
		return l, nil
	}
	return 0, nil
}

func (w *pwsetter) Write(p []byte) (int, error) {
	if bytes.Equal(p, []byte(`Password: `)) {
		return len(p), nil
	}
	if bytes.Equal(p, []byte(` Confirm: `)) {
		return len(p), nil
	}
	return len(p), nil
}

func (a admin) SetPassword(forUser vocab.IRI, pw string) SSHCmd {
	pwSetter := new(pwsetter)
	pwSetter.pw = pw
	return SSHCmd{
		Cmd:  []string{"accounts", "pass", string(forUser)},
		User: string(a.user),
		Key:  a.prv,
		IO:   pwSetter,
	}
}

func (a admin) ExtractOAuth2Bearer(user vocab.IRI, tok *c2s.BearerSigner) SSHCmd {
	tokExtr := new(oauthextractorrw)
	tokExtr.Tok = tok
	return SSHCmd{
		Cmd:  []string{"oauth", "token", "add", string(user)},
		User: string(a.user),
		Key:  a.prv,
		IO:   tokExtr,
	}
}
