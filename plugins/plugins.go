package plugins

import (
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/eguevara/prow-lite/gitlab"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
)

var (
	mergeCommentEventHandlers = map[string]MergeCommentEventHandler{}
)

// MergeCommentEventHandler is a function type that is used as the value of the
// map to run the plugin handler code.
type MergeCommentEventHandler func(PluginClient, gitlab.MergeCommentEvent) error

// RegisterMergeCommentEventHandler will register to a map with the key is the
// pluginName and the value is a function that handles the plugin code.
func RegisterMergeCommentEventHandler(name string, fn MergeCommentEventHandler) {
	mergeCommentEventHandlers[name] = fn
}

// PluginClient may be used concurrently, so each entry must be thread-safe.
type PluginClient struct {
	// PluginConfig provides plugin-specific options
	PluginConfig *Configuration

	Logger *logrus.Entry

	GitlabClient *gitlab.Client
}

// PluginAgent manages how handler communicates with other clients and handles
// reading plugin configrations files.
type PluginAgent struct {
	PluginClient

	mut           sync.Mutex
	configuration *Configuration
}

// Config returns the plugins configuration reference.
func (pa *PluginAgent) Config() *Configuration {
	pa.mut.Lock()
	defer pa.mut.Unlock()
	return pa.configuration
}

// Load attempts to load config from the path. It returns an error if either
// the file can't be read or it contains an unknown plugin.
func (pa *PluginAgent) Load(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	np := &Configuration{}
	if err := yaml.Unmarshal(b, np); err != nil {
		return err
	}

	if len(np.Plugins) == 0 {
		logrus.Warn("no plugins specified-- check syntax?")
	}

	pa.Set(np)
	return nil
}

// Set attempts to set the plugins that are enabled on repos. Plugins are listed
// as a map from repositories to the list of plugins that are enabled on them.
// Specifying simply an org name will also work, and will enable the plugin on
// all repos in the org.
func (pa *PluginAgent) Set(pc *Configuration) {
	pa.mut.Lock()
	defer pa.mut.Unlock()
	pa.configuration = pc
}

// Start starts polling path for plugin config. If the first attempt fails,
// then start returns the error. Future errors will halt updates but not stop.
func (pa *PluginAgent) Start(path string) error {
	if err := pa.Load(path); err != nil {
		return err
	}
	ticker := time.Tick(1 * time.Minute)

	go func() {
		for range ticker {
			if err := pa.Load(path); err != nil {
				logrus.WithField("path", path).WithError(err).Error("Error loading plugin config.")
			}
		}
	}()
	return nil
}

// MergeCommentEventHandlers returns a map of plugin names to handlers for the
// repo. When a plugin is loading, its init() function registers itself to the
// pluginAgent as a map where the plugin name is the key and the value is a
// function type.
func (pa *PluginAgent) MergeCommentEventHandlers(owner, repo string) map[string]MergeCommentEventHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]MergeCommentEventHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := mergeCommentEventHandlers[p]; ok {
			hs[p] = h
		}
	}

	return hs
}

// getPlugins returns a list of plugins that are enabled on a given (gitlab group
// or repository). Plugins are enabled in the plugins.yaml file specified by the
// -plugin-config flag.
func (pa *PluginAgent) getPlugins(owner, repo string) []string {
	var plugins []string

	fullName := fmt.Sprintf("%s/%s", owner, repo)
	plugins = append(plugins, pa.configuration.Plugins[owner]...)
	plugins = append(plugins, pa.configuration.Plugins[fullName]...)

	return plugins
}

// Configuration is used to store the data structure of the plugin.yaml file.
type Configuration struct {
	Plugins map[string][]string `json:"plugins,omitempty"`
}
