package main

import (
	"context"
	"flag"
	"fmt"
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
	gitCommitHash  = "local"
	configPathFlag = flag.String("config-path", "", "REQUIRED")
)

type Config struct {
	// REQUIRED: Address to listen on
	Addr string `yaml:"addr"`

	// Redis credentials are REQUIRED for production
	// RedisPassword is set via REDIS_PASSWORD env var
	// while RedisUrl is expected in the config file.
	//
	// If RedisUrl || RedisPassword are empty, miniredis will be used.
	RedisUrl      string `yaml:"redis-url"`
	redisPassword string

	// REQUIRED: Path to private key
	KeyPath string `yaml:"key-path"`

	// REQUIRED: Map of app tokens to app secrets
	AppTokensAndSecrets map[string]string `yaml:"app-tokens-and-secrets"`

	// OPTIONAL: Whether to log more information
	Verbose bool `yaml:"verbose"`

	// OPTIONAL: Logged in the "version" field of the any endpoint
	SdkVersion string `yaml:"sdk-version"`
}

func main() {
	logrus.SetReportCaller(true)
	if err := mainErr(); err != nil {
		logrus.Fatalf(err.Error())
	}
}

func initAndParseConfig() (config *Config, onExit func(), err error) {
	flag.Parse()

	b, err := os.ReadFile(*configPathFlag)
	if err != nil {
		return nil, nil, errors.Wrap(err, "")
	}
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "")
	}

	if config.KeyPath == "" {
		return nil, nil, errors.New("key-path is empty")
	}
	if strings.HasPrefix(config.KeyPath, "./") {
		config.KeyPath = filepath.Join(projectpath.Root, config.KeyPath)
	}

	config.redisPassword = os.Getenv("REDIS_PASSWORD")
	if config.RedisUrl == "" || config.redisPassword == "" {
		logrus.Infof(
			"redis-url or REDIS_PASSWORD flags are empty. Using miniredis...")
		config.redisPassword = ""
		m, err := miniredis.Run()
		if err != nil {
			return nil, nil, errors.Wrap(err, "")
		}
		config.RedisUrl = m.Addr()
		onExit = func() {
			if m != nil {
				m.Close()
			}
		}
	}
	return config, onExit, nil
}

func mainErr() error {
	// Init config
	config, onExit, err := initAndParseConfig()
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
	fmt.Printf("redis-url: %s\n", config.RedisUrl)
	fmt.Printf("redis-password: %s\n", config.redisPassword)
	rdw := &rediswrapper.RedisWrapper{
		Client: redis.NewClient(&redis.Options{
			Addr:     config.RedisUrl,
			Password: config.redisPassword,
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
		// TODO <27-02-22, afjoseph> Definitely fix the corsOriginWhileList
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
