package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	configPathFlag = flag.String("config-path", "", "REQUIRED")
	gitCommitHash  = "unknown"
)

type Config struct {
	Addr                 string `yaml:"addr"`
	PlisskenAppSecret    string `yaml:"plissken-app-secret"`
	PlisskenAppToken     string `yaml:"plissken-app-token"`
	PlisskenAuthEndpoint string `yaml:"plissken-auth-endpoint"`
	Verbose              bool   `yaml:"verbose"`
}

func parseFlags() (config *Config, err error) {
	flag.Parse()
	b, err := os.ReadFile(*configPathFlag)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return config, nil
}

func main() {
	// Parse flags
	config, err := parseFlags()
	if err != nil {
		panic(err)
	}

	// Init logger
	logrus.SetReportCaller(true)
	if config.Verbose {
		logrus.SetLevel(logrus.TraceLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	// Init Redis
	m, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer m.Close()

	// Define handlers
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, gitCommitHash)
		w.WriteHeader(http.StatusOK)
	})
	http.HandleFunc("/get-resource", func(w http.ResponseWriter, r *http.Request) {
		// CORS stuff
		setupCORS(w, r)
		if r.Method == "OPTIONS" {
			return
		}

		// Check credentials
		username := r.URL.Query().Get("username")
		sessionToken := r.URL.Query().Get("session_token")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err = checkCredentials(
			ctx,
			config.PlisskenAppSecret,
			config.PlisskenAppToken,
			config.PlisskenAuthEndpoint,
			username,
			sessionToken)
		if err != nil {
			logrus.Errorf("while checking credentials: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Get resource
		val, err := m.Get(fmt.Sprintf("%s:last_haircut", username))
		if err != nil {
			logrus.Errorf("while getting value: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, val)
	})

	http.HandleFunc("/put-resource", func(w http.ResponseWriter, r *http.Request) {
		// CORS stuff
		setupCORS(w, r)
		if r.Method == "OPTIONS" {
			return
		}

		// Check credentials
		username := r.URL.Query().Get("username")
		sessionToken := r.URL.Query().Get("session_token")
		ts := r.URL.Query().Get("ts")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err = checkCredentials(
			ctx,
			config.PlisskenAppSecret,
			config.PlisskenAppToken,
			config.PlisskenAuthEndpoint,
			username,
			sessionToken)
		if err != nil {
			logrus.Errorf("while checking credentials: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Put resource
		err = m.Set(
			fmt.Sprintf("%s:last_haircut", username),
			ts)
		if err != nil {
			logrus.Errorf("while putting value: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// Start server
	if err = http.ListenAndServe(config.Addr, nil); err != nil {
		panic(err)
	}
}

type PlisskenCheckCredentialsResponseData struct {
	Username   string `json:"username"`
	CreatedAt  int64  `json:"created_at"`
	SdkVersion string `json:"sdk_version"`
	ExpiresAt  int64  `json:"expires_at"`
}

func checkCredentials(
	ctx context.Context,
	plisskenAppSecret,
	plisskenAppToken,
	plisskenEndpoint,
	username, sessionToken string) (*PlisskenCheckCredentialsResponseData, error) {
	req, err := http.NewRequestWithContext(
		ctx, "GET", plisskenEndpoint+"/check-credentials",
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	q := req.URL.Query()
	q.Add("apptoken", plisskenAppToken)
	q.Add("appsecret", plisskenAppSecret)
	q.Add("username", username)
	q.Add("session_token", sessionToken)
	req.URL.RawQuery = q.Encode()

	logrus.Debugf("Running /check-credentials request for %s on %s",
		username, plisskenEndpoint)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	typedResp := &PlisskenCheckCredentialsResponseData{}
	err = json.NewDecoder(resp.Body).Decode(typedResp)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	defer resp.Body.Close()

	logrus.Debugf("Received /check-credentials request for %s on %s",
		username, plisskenEndpoint)
	return typedResp, nil
}

func setupCORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type")
}
