package config

import (
	"fmt"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/joho/godotenv"
	"os"
	"strconv"
	"strings"
)


type backendConfig struct {
	Enabled bool
	Host    string
	Port    int64
	User    string
	Pw      string
	Name    string
}

type Options struct {
	Env                 env.Type
	LogLevel            log.Level
	Secure              bool
	Host                string
	BaseURL             string
	DB                  backendConfig
}

func LoadFromEnv() (Options, error) {
	conf := Options{}

	conf.Env = env.ValidTypeOrDev(os.Getenv("ENV"))
	configs := []string{
		".env",
		fmt.Sprintf(".env.%s", conf.Env),
	}

	lvl := os.Getenv("LOG_LEVEL")
	switch strings.ToLower(lvl) {
	case "trace":
		conf.LogLevel = log.TraceLevel
	case "debug":
		conf.LogLevel = log.DebugLevel
	case "warn":
		conf.LogLevel = log.WarnLevel
	case "error":
		conf.LogLevel = log.ErrorLevel
	case "info":
		fallthrough
	default:
		conf.LogLevel = log.InfoLevel
	}

	for _, f := range configs {
		if err := godotenv.Overload(f); err != nil {
			return conf, err
		}
	}

	if conf.Host == "" {
		conf.Host = os.Getenv("HOSTNAME")
	}
	conf.Secure, _ = strconv.ParseBool(os.Getenv("HTTPS"))
	if conf.Secure {
		conf.BaseURL = fmt.Sprintf("https://%s", conf.Host)
	} else {
		conf.BaseURL = fmt.Sprintf("http://%s", conf.Host)
	}

	conf.DB.Host = os.Getenv("DB_HOST")
	conf.DB.Pw = os.Getenv("DB_PASSWORD")
	conf.DB.Name = os.Getenv("DB_NAME")
	var err error
	if conf.DB.Port, err = strconv.ParseInt(os.Getenv("DB_PORT"), 10, 32); err != nil {
		conf.DB.Port = 5432
	}

	conf.DB.User = os.Getenv("DB_USER")

	return conf, nil
}
