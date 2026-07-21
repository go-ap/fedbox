package fedbox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/exec"
	"golang.org/x/crypto/ssh"
)

type fboxContainer struct {
	img fboxImage
	tc.Container
}

func (fc fboxContainer) Exec(ctx context.Context, cmd []string, opts ...exec.ProcessOption) (int, io.Reader, error) {
	if cmd[0] == ctlBin {
		// NOTE(marius): if the command actually wants to call the "fedbox" binary,
		// we execute it using docker exec.
		// Otherwise, we treat this as it needs to be run though SSH.
		return fc.Container.Exec(ctx, cmd, opts...)
	}
	conf := exec.ProcessOptions{}
	for _, opt := range opts {
		opt.Apply(&conf)
	}

	// NOTE(marius): extract the host we need to use for SSH connections
	sshHost, _ := fc.Host(ctx)
	portsMap, err := fc.Ports(ctx)
	if err != nil {
		return 0, nil, err
	}
	for label, ports := range portsMap {
		if label.Num() == uint16(fc.img.conf.SSHPort) && len(ports) == 1 {
			sshHost = net.JoinHostPort(sshHost, ports[0].HostPort)
		}
	}

	// NOTE(marius): extract authorization mechanisms from env variables
	initFns := make([]ssh.AuthMethod, 0)
	if fc.img.pw != nil {
		initFns = append(initFns, ssh.Password(string(fc.img.pw)))
	}
	if fc.img.key != nil {
		sig, err := ssh.NewSignerFromKey(fc.img.key)
		if err != nil {
			return 0, nil, err
		}
		initFns = append(initFns, ssh.PublicKeys(sig))
	}
	if len(initFns) == 0 {
		return 0, nil, fmt.Errorf("no SSH authorization methods found")
	}
	if len(fc.img.rootIRI) == 0 {
		return 0, nil, fmt.Errorf("no user for SSH authorization found")
	}
	config := &ssh.ClientConfig{
		User:            conf.ExecConfig.User,
		Auth:            initFns,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	client, err := ssh.Dial("tcp", sshHost, config)
	if err != nil {
		return 0, nil, fmt.Errorf("unable to connect to SSH server %s %w", sshHost, err)
	}
	// Create a session. It is one session per command.
	session, err := client.NewSession()
	if err != nil {
		return 0, nil, err
	}
	defer session.Close()
	errBuff := bytes.Buffer{}
	session.Stderr = &errBuff

	outBuff := bytes.Buffer{}
	if w, ok := conf.Reader.(io.Writer); ok {
		session.Stdout = io.MultiWriter(&outBuff, w)
	}
	session.Stdin = conf.Reader

	if err = session.Run(strings.Join(cmd, " ")); err != nil {
		if errBuff.Len() == 0 {
			return 0, nil, fmt.Errorf("command execution failed: %w", err)
		}
		return 0, nil, fmt.Errorf("command execution failed: %w\n%s", err, errBuff.String())
	}
	// NOTE(marius): if stdErr had output, we treat it as an error
	if errBuff.Len() > 0 {
		return 0, nil, STDErr(errBuff)
	}
	return outBuff.Len(), &outBuff, nil
}

const ctlBin = "fedbox"

type STDErr bytes.Buffer

func (e STDErr) Error() string {
	b := bytes.Buffer(e)
	return b.String()
}
