package integration

import (
	"flag"
	"os"
	"testing"
)

var Verbose bool

func TestMain(m *testing.M) {
	flag.BoolVar(&Verbose, "verbose", false, "enable more verbose logging")
	flag.Parse()

	//name, err := buildImage(context.Background())
	//if err != nil {
	//	fmt.Fprintf(os.Stderr, "error building: %s", err)
	//	os.Exit(-1)
	//}
	//fmt.Fprintf(os.Stdout, "built image: %s", name)

	if st := m.Run(); st != 0 {
		os.Exit(st)
	}
}
