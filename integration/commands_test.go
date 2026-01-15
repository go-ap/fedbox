package integration

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/go-ap/errors"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/google/go-cmp/cmp"
	tc "github.com/testcontainers/testcontainers-go"
)

var equateWeakErrors = cmp.FilterValues(areErrors, cmp.Comparer(compareErrors))

func areErrors(x, y interface{}) bool {
	_, ok1 := x.(error)
	_, ok2 := y.(error)
	return ok1 && ok2
}

func compareErrors(x, y interface{}) bool {
	xe := x.(error)
	ye := y.(error)
	return errors.Is(xe, ye) || errors.Is(ye, xe) || xe.Error() == ye.Error()
}

func eqErrs(want, got error) bool {
	return cmp.Equal(want, got, equateWeakErrors)
}

func diffErrs(want, got error) string {
	return cmp.Diff(want, got, equateWeakErrors)
}

func Test_Commands_inSeparateContainers(t *testing.T) {
	toRun := []struct {
		Name       string
		Host       string
		Cmd        tc.Executable
		WantOutput string
		WantErr    error
	}{
		{
			Name: "--help",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"reload"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			// NOTE(marius): this is strange. The output should actually be the
			WantOutput: "FedBOX SSH: OK\n",
		},
		{
			Name: "reload",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"reload"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			WantOutput: "FedBOX SSH: OK\n",
		},
		{
			Name: "maintenance",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"maintenance"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			WantOutput: "FedBOX SSH: OK\n",
		},
		{
			Name: "stop",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"stop"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			WantOutput: "FedBOX SSH: OK\n",
		},
		{
			Name: "stop",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"stop"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			WantOutput: "FedBOX SSH: OK\n",
		},
	}

	for _, test := range toRun {
		var c2sFedBOX = c.C2SfedBOX(
			c.WithEnv(defaultC2SEnv),
			c.WithImageName(fedBOXImageName),
			c.WithKey(defaultPrivateKey),
			c.WithUser(service.ID),
			c.WithPw(defaultPassword),
		)

		images := c.Suite{c2sFedBOX}

		ctx := context.Background()
		cont, err := c.Init(ctx, t, images...)
		if err != nil {
			t.Fatalf("unable to initialize containers: %s", err)
		}

		t.Cleanup(func() {
			cont.Cleanup(t)
		})

		t.Run(test.Name, func(t *testing.T) {
			out, err := cont.RunCommand(ctx, test.Host, test.Cmd)
			if !eqErrs(test.WantErr, err) {
				if test.Cmd == nil {
					t.Fatalf("Err received executing nil command %s: %+v", test.Host, diffErrs(test.WantErr, err))
				}
				t.Fatalf("Err received executing command %s->%v: %+v", test.Host, test.Cmd.AsCommand(), diffErrs(test.WantErr, err))
			}
			if len(test.WantOutput) > 0 && out == nil {
				t.Fatalf("No output from command when it was expected %s->%v, size %d", test.Host, test.Cmd.AsCommand(), len(test.WantOutput))
			}
			raw, _ := io.ReadAll(out)
			if !bytes.Equal([]byte(test.WantOutput), raw) {
				t.Errorf("Output from command differs %s->%v\n %s", test.Host, test.Cmd.AsCommand(), cmp.Diff(test.WantOutput, string(raw)))
			}
		})
	}
}
