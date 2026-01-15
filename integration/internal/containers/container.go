package containers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/exec"
	"golang.org/x/crypto/ssh"
)

type fboxContainer struct {
	tc.Container
}

func (fc fboxContainer) Exec(ctx context.Context, cmd []string, opts ...exec.ProcessOption) (int, io.Reader, error) {
	r, err := execSSH(ctx, fc.Container, cmd, opts...)
	return 0, r, err
}

const ctlBin = "fedbox"

func execSSH(ctx context.Context, fc tc.Container, cmd []string, opts ...exec.ProcessOption) (io.Reader, error) {
	if cmd[0] == ctlBin {
		// NOTE(marius): if the command actually wants to call the "fedbox" binary,
		// we execute it using the docker exec mechanism
		// Otherwise, we treat this as a command that needs to be run IN SSH.
		_, out, err := fc.Exec(ctx, cmd, opts...)
		return out, err
	}

	host, _ := fc.Host(ctx)
	portsMap, err := fc.Ports(ctx)
	if err != nil {
		return nil, err
	}
	conf := exec.ProcessOptions{}
	for _, opt := range opts {
		opt.Apply(&conf)
	}

	for label, ports := range portsMap {
		if strings.HasPrefix(string(label), "4044") && len(ports) > 0 {
			host = net.JoinHostPort(host, ports[0].HostPort)
		}
	}

	// NOTE(marius): extract authorization mechanisms from env variables
	initFns := make([]ssh.AuthMethod, 0)
	for _, env := range conf.ExecConfig.Env {
		if strings.HasPrefix(env, "_PK=") {
			prv, err := ssh.ParseRawPrivateKey([]byte(strings.TrimPrefix(env, "_PK=")))
			if err != nil {
				continue
			}
			sig, err := ssh.NewSignerFromKey(prv)
			if err != nil {
				continue
			}
			initFns = append(initFns, ssh.PublicKeys(sig))
		}
		if strings.HasPrefix(env, "_PW=") {
			pw := strings.TrimPrefix(env, "_PW=")
			initFns = append(initFns, ssh.Password(pw))
		}
	}
	if len(initFns) == 0 {
		return nil, fmt.Errorf("no SSH authorization methods found")
	}
	if len(conf.ExecConfig.User) == 0 {
		return nil, fmt.Errorf("no user for SSH authorization found")
	}
	config := &ssh.ClientConfig{
		User:            conf.ExecConfig.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth:            initFns,
	}

	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return nil, err
	}
	// Create a session. It is one session per command.
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()
	errBuff := bytes.Buffer{}
	outBuff := bytes.Buffer{}
	session.Stderr = &errBuff
	session.Stdout = &outBuff
	session.Stdin = conf.Reader

	if err = session.Run(strings.Join(cmd, " ")); err != nil {
		if errBuff.Len() == 0 {
			return nil, fmt.Errorf("command execution failed: %w", err)
		}
		return nil, fmt.Errorf("command execution failed: %w\n%s", err, errBuff.String())
	}
	// NOTE(marius): if stdErr had output, we treat it as an error
	if errBuff.Len() > 0 {
		return nil, STDErr(errBuff)
	}
	return &outBuff, nil
}

type STDErr bytes.Buffer

func (e STDErr) Error() string {
	b := bytes.Buffer(e)
	return b.String()
}
