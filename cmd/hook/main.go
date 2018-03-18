package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/eguevara/prow-lite/api"
	"github.com/eguevara/prow-lite/config"
	"github.com/eguevara/prow-lite/gitlab"
	"github.com/eguevara/prow-lite/logrusutil"
	"github.com/eguevara/prow-lite/plugins"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

const (
	version = "1.0.0"
)

type options struct {
	address      string
	pluginConfig string
	configPath   string
}

func getOptions() options {
	o := options{}
	flag.StringVar(&o.address, "listen-address", "0.0.0.0:3000", "Address to listen web to.")
	flag.StringVar(&o.pluginConfig, "plugin-config", "../../plugins.yaml", "Path to plugins.yaml.")
	flag.StringVar(&o.configPath, "config-path", "../../config.yaml", "Path to config.yaml.")

	flag.Parse()

	return o
}
func main() {

	// Set up application configuration.
	cfg := getOptions()

	logrus.SetFormatter(logrusutil.NewDefaultFieldsFormatter(nil, logrus.Fields{"component": "hook"}))

	// Config Agent loads the config.yaml file for application configurations
	// and continues to poll the file for changes.
	configAgent := &config.Agent{}
	if err := configAgent.Start(cfg.configPath); err != nil {
		logrus.WithError(err).Fatal("Error starting config agent.")
	}

	// Get config settings from config.yaml
	shutdownTimeout := configAgent.Config().ShutdownTimeout
	gitlabToken := configAgent.Config().Token
	gitlabBaseURL := configAgent.Config().BaseURL

	// Sets up the gitlab client for communicating to the gitlab api service.
	var gitlabClient *gitlab.Client
	gitlabClient, err := gitlab.NewClient(gitlabToken, gitlabBaseURL)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting gitlab client.")
	}

	pluginAgent := &plugins.PluginAgent{}
	pluginAgent.PluginClient = plugins.PluginClient{
		Logger:       logrus.WithField("agent", "plugin"),
		GitlabClient: gitlabClient,
	}

	// Start will load the plugin config file and continue to poll the file for changes.
	if err := pluginAgent.Start(cfg.pluginConfig); err != nil {
		logrus.WithError(err).Fatal("Error starting plugins.")
	}

	handler := &api.HooksHandler{
		HMACSecret: []byte("abcde12345"),
		Plugins:    pluginAgent,
	}

	// Using gorilla mux for richer routing
	r := mux.NewRouter()
	r.Handle("/hook", handler).Methods("POST")

	// Create a new server and set timeout values.
	server := http.Server{
		Addr:           cfg.address,
		Handler:        r,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Start the listener.
	go func() {
		logrus.Printf("start: Listening on %s\n", cfg.address)
		logrus.Println("start: Process ID", os.Getpid())

		if err := server.ListenAndServe(); err != nil {
			logrus.Println("ListenAndServe returns an error", err)
		}
	}()

	// Listen for an interrupt signal from the OS.
	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, os.Interrupt)

	// Wait for a signal to shutdown.
	logrus.Printf("shutdown: Signal %v", <-signalChan)

	// Create a context to attempt a graceful 5 second shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(shutdownTimeout)*time.Second)
	defer cancel()

	// Attempt the graceful shutdown by closing the listener and
	// completing all inflight requests.
	if err := server.Shutdown(ctx); err != nil {
		logrus.Printf("shutdown : Graceful shutdown did not complete in %v : %v", time.Duration(shutdownTimeout)*time.Second, err)

		// Looks like we timedout on the graceful shutdown. Kill it hard.
		if err := server.Close(); err != nil {
			logrus.Printf("shutdown : Error killing server : %v", err)
		}
	}

	logrus.Println("shutdown: Graceful complete")
}
