package lgtm

import (
	"github.com/eguevara/prow-lite/gitlab"
	"github.com/eguevara/prow-lite/plugins"
	"github.com/sirupsen/logrus"
)

const pluginName = "lgtm"

var (
	lgtmLabel = "lgtm"
)

func init() {
	plugins.RegisterMergeCommentEventHandler(pluginName, handleMergeComment)
}

func handleMergeComment(pc plugins.PluginClient, e gitlab.MergeCommentEvent) error {
	pc.Logger.Debugln("handleMergeComment")
	return handle(pc.GitlabClient, pc.Logger, &e)
}

func handle(gc *gitlab.Client, log *logrus.Entry, e *gitlab.MergeCommentEvent) error {
	log.Debugln("handle")
	labels, err := gc.GetMergeRequestLabels(e.ObjectAttributes.ProjectID, e.MergeRequest.IID)
	if err != nil {
		log.WithError(err).Errorf("Failed to get the labels on %d#%d.", e.ObjectAttributes.ProjectID, e.MergeRequest.IID)
	}
	hasLGTM := false
	for _, l := range labels {
		if l == lgtmLabel {
			hasLGTM = true
			break
		}
	}

	if !hasLGTM {
		labels = append(labels, lgtmLabel)
		gc.AddLabel(e.ObjectAttributes.ProjectID, e.MergeRequest.IID, labels)
		log.Debugln("label added!!")
	}

	return nil
}
