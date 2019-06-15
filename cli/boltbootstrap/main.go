package main

import (
	"flag"
	"fmt"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/storage/boltdb"
)

func errf(s string, par ...interface{}) {
	fmt.Printf(s, par...)
}

func main() {
	var environ string

	flag.StringVar(&environ, "env", string(env.DEV), "environment")
	flag.Parse()

	conf, err := config.LoadFromEnv(environ)
	if err != nil {
		errf("Unable to load settings from environment variables: %s", err)
	}
	err = boltdb.Bootstrap(conf.BoltDBPath, []byte(conf.Host), conf.BaseURL)
	if err != nil {
		errf("Unable to update bolt db: %s", err)
	}
	fmt.Println("OK")
}
