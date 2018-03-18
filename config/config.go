package config

import (
	"io/ioutil"
	"sync"
	"time"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
)

// Config stores all configuration options for app.
type Config struct {
	// LogLevel enables dynamically updating the log level of the
	// standard logger that is used by all prow components.
	//
	// Valid values:
	//
	// "debug", "info", "warn", "warning", "error", "fatal", "panic"
	//
	// Defaults to "info".
	LogLevel string `json:"log_level,omitempty"`

	// Token is the gitlab token used for authentication.
	Token string `json:"token,omitempty"`

	// BaseURL is the base url for the gitlab service.
	BaseURL string `json:"base_url,omitempty"`

	// ShutdownTimeout is the HTTP setting used to manage shutdown settings on
	// the server.
	ShutdownTimeout int64 `json:"shutdown_timeout,omitempty"`
}

// Agent manages how handler communicates with other clients and handles
// reading plugin configrations files.
type Agent struct {
	mut           sync.Mutex
	configuration *Config
}

// Config returns the plugins configuration reference.
func (ca *Agent) Config() *Config {
	ca.mut.Lock()
	defer ca.mut.Unlock()
	return ca.configuration
}

// Load attempts to load config from the path. It returns an error if either
// the file can't be read or it contains an unknown plugin.
func (ca *Agent) Load(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	np := &Config{}
	if err := yaml.Unmarshal(b, np); err != nil {
		return err
	}

	// Set up Logging defaults
	if np.LogLevel == "" {
		np.LogLevel = "info"
	}
	lvl, err := logrus.ParseLevel(np.LogLevel)
	if err != nil {
		return err
	}
	logrus.SetLevel(lvl)

	ca.Set(np)
	return nil
}

// Set sets the data structure to the configuration field of the agent.
func (ca *Agent) Set(pc *Config) {
	ca.mut.Lock()
	defer ca.mut.Unlock()
	ca.configuration = pc
}

// Start starts polling path for config. If the first attempt fails,
// then start returns the error. Future errors will halt updates but not stop.
func (ca *Agent) Start(path string) error {
	if err := ca.Load(path); err != nil {
		return err
	}
	ticker := time.Tick(1 * time.Minute)

	go func() {
		for range ticker {
			if err := ca.Load(path); err != nil {
				logrus.WithField("path", path).WithError(err).Error("Error loading plugin config.")
			}
		}
	}()
	return nil
}
