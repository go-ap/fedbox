package cmd

import (
	"bytes"
	"fmt"
	"github.com/go-ap/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/urfave/cli.v2"
	"os"
)

var ctl Control
var logger = logrus.New()

func Before(c *cli.Context) error {
	logger.Level = logrus.ErrorLevel
	ct, err := setup(c, logger)
	if err == nil {
		ctl = *ct
	}

	return err
}

func loadPwFromStdin(confirm bool, s string, params ...interface{}) ([]byte, error) {
	fmt.Printf(s+" pw: ", params...)
	pw1, _ := terminal.ReadPassword(0)
	fmt.Println()
	if confirm {
		fmt.Printf("pw again: ")
		pw2, _ := terminal.ReadPassword(0)
		fmt.Println()
		if !bytes.Equal(pw1, pw2) {
			return nil, errors.Errorf("Passwords do not match")
		}
	}
	return pw1, nil
}

func Errf(s string, par ...interface{}) {
	fmt.Fprintf(os.Stderr, s, par...)
}
