package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/eguevara/prow-lite/gitlab"
	"github.com/eguevara/prow-lite/plugins"
	"github.com/eguevara/prow-lite/utils"
	"github.com/sirupsen/logrus"
)

// HooksHandler handles all request for the /hooks endpoint.
type HooksHandler struct {
	Plugins    *plugins.PluginAgent
	HMACSecret []byte
}

// HooksResponse stores the version handler response.
type HooksResponse struct {
	Status string `json:"status"`
}

func (h *HooksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, payload, ok := ValidateWebhook(w, r, h.HMACSecret)
	if !ok {
		return
	}

	response := HooksResponse{
		Status: "ok",
	}

	utils.Respond(w, response, nil)

	if err := h.processEvent(eventType, payload, r.Header); err != nil {
		logrus.WithError(err).Error("Error parsing event.")
	}
}

// ValidateWebhook ensures that the provided request conforms to the
// format of a Github webhook and the payload can be validated with
// the provided hmac secret. It returns the event type, the event guid,
// the payload of the request, and whether the webhook is valid or not.
func ValidateWebhook(w http.ResponseWriter, r *http.Request, hmacSecret []byte) (string, []byte, bool) {
	defer r.Body.Close()

	// Our health check uses GET, so just kick back a 200.
	if r.Method == http.MethodGet {
		return "", nil, false
	}

	eventType := r.Header.Get("X-Gitlab-Event")
	if eventType == "" {
		resp := "400 Bad Request: Missing X-GitHub-Event Header"
		logrus.Debug(resp)
		http.Error(w, resp, http.StatusBadRequest)
		return "", nil, false
	}

	contentType := r.Header.Get("content-type")
	if contentType != "application/json" {
		resp := "400 Bad Request: Hook only accepts content-type: application/json - please reconfigure this hook on GitHub"
		logrus.Debug(resp)
		http.Error(w, resp, http.StatusBadRequest)
		return "", nil, false
	}

	// requestDump, err := httputil.DumpRequest(r, true)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Println(string(requestDump))

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		resp := "500 Internal Server Error: Failed to read request body"
		logrus.Debug(resp)
		http.Error(w, resp, http.StatusInternalServerError)
		return "", nil, false
	}

	return eventType, payload, true
}

func (h *HooksHandler) processEvent(eventType string, payload []byte, header http.Header) error {
	l := logrus.WithFields(
		logrus.Fields{
			"event-type": eventType,
		},
	)

	switch eventType {
	case "Note Hook":
		l.Infoln("Node Hook EventType")
		var mc gitlab.MergeCommentEvent
		if err := json.Unmarshal(payload, &mc); err != nil {
			return err
		}

		h.handleMergeCommentEvent(l, mc)
	}

	l.Println("processEvent")
	return nil
}

func (h *HooksHandler) handleMergeCommentEvent(l *logrus.Entry, mc gitlab.MergeCommentEvent) {
	l = l.WithFields(logrus.Fields{
		"repo":      mc.Repository.Name,
		"commenter": mc.User.Username,
		"type":      mc.ObjectAttributes.NoteableType,
	})

	// Iterate through each registered Handler with the gitlab group or repo map
	// key.  Handlers are registered when the plugin is loaded (init())
	for p, handler := range h.Plugins.MergeCommentEventHandlers(mc.User.Username, mc.Repository.Name) {

		// An anonymous function that takes in the plugin (key) and the handler
		// (value) of the map returned from the for-each. It then creates a new
		// pluginClient and calls the handler function for the plugin to perform
		// the logic of the plugin. Run as a goroutine.
		go func(p string, handler plugins.MergeCommentEventHandler) {
			pc := h.Plugins.PluginClient
			pc.Logger = l.WithField("plugin", p)
			pc.PluginConfig = h.Plugins.Config()
			if err := handler(pc, mc); err != nil {
				pc.Logger.WithError(err).Error("Error handling PushEvent.")
			}
		}(p, handler)
	}
}
