package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/afjoseph/plissken-auth-server/projectpath"
	"github.com/afjoseph/plissken-auth-server/rediswrapper"
	"github.com/afjoseph/plissken-auth-server/server"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	gitCommitHash  = "unknown"
	configPathFlag = flag.String("config-path", "", "REQUIRED")
)

type Config struct {
	Addr                string            `yaml:"addr"`
	RedisUrl            string            `yaml:"redis-url"`
	RedisPassword       string            `yaml:"redis-password"`
	KeyPath             string            `yaml:"key-path"`
	AppTokensAndSecrets map[string]string `yaml:"app-tokens-and-secrets"`
	Verbose             bool              `yaml:"verbose"`
	SdkVersion          string            `yaml:"sdk-version"`
}

func main() {
	logrus.SetReportCaller(true)
	if err := mainErr(); err != nil {
		logrus.Fatalf(err.Error())
	}
}

func parseFlags() (config *Config, err error, onExit func()) {
	flag.Parse()

	b, err := os.ReadFile(*configPathFlag)
	if err != nil {
		return nil, errors.Wrap(err, ""), nil
	}
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return nil, errors.Wrap(err, ""), nil
	}

	if config.KeyPath == "" {
		return nil, errors.New("key-path is empty"), nil
	}
	if strings.HasPrefix(config.KeyPath, "./") {
		config.KeyPath = filepath.Join(projectpath.Root, config.KeyPath)
	}

	if config.RedisUrl == "" {
		logrus.Infof(
			"redis-url flag is empty. Using miniredis...")
		config.RedisPassword = ""
		m, err := miniredis.Run()
		if err != nil {
			return nil, errors.Wrap(err, ""), nil
		}
		config.RedisUrl = m.Addr()
		onExit = func() {
			if m != nil {
				m.Close()
			}
		}
	}
	config.RedisPassword = os.Getenv("REDIS_PASSWORD")
	if config.RedisPassword == "" {
		return nil, errors.New("REDIS_PASSWORD is empty"), nil
	}
	return config, nil, onExit
}

func mainErr() error {
	config, err, onExit := parseFlags()
	if err != nil {
		return errors.Wrap(err, "")
	}
	defer func() {
		if onExit != nil {
			onExit()
		}
	}()
	if config.Verbose {
		logrus.SetLevel(logrus.TraceLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	// Init RedisWrapper
	rdw := &rediswrapper.RedisWrapper{
		Client: redis.NewClient(&redis.Options{
			Addr:     config.RedisUrl,
			Password: config.RedisPassword,
			DB:       0,
		})}

	// Add all app tokens and secrets to redis
	for appToken, appSecret := range config.AppTokensAndSecrets {
		err = rdw.StoreAppSecret(context.Background(), appToken, appSecret)
		if err != nil {
			return errors.Wrap(err, "")
		}
	}

	// Read key from file
	serverPrivateKey, err := os.ReadFile(config.KeyPath)
	if err != nil {
		return errors.Wrap(err, "")
	}
	errChan := make(chan error)
	srv, err := server.Host(
		serverPrivateKey,
		// XXX <27-02-22, afjoseph> Keep corsOriginWhileList nil since we'll
		// never use this server in production
		nil,
		config.Addr,
		config.Verbose,
		config.SdkVersion,
		gitCommitHash,
		rdw,
		errChan)
	if err != nil {
		return errors.Wrap(err, "")
	}
	logrus.Infof("Running server on localhost:%v", srv.Port)

	// Wait for an error or an interrupt to occur.
	// If an interrupt is hit, shut-down server gracefully.
	// If that fails, force-exit
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt)
	select {
	case err := <-errChan:
		logrus.Errorf("Error while serving: %v\n", errors.Wrap(err, ""))
	case <-stopChan:
		logrus.Infoln("Stop signal received")
	}

	logrus.Infoln("Attempting to shut-down gracefully...")
	ctx, cancel := context.WithTimeout(
		context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}
