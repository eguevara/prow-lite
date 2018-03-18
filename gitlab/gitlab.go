package gitlab

import (
	"github.com/sirupsen/logrus"
	gl "github.com/xanzy/go-gitlab"
)

// Client represents the client connection.
type Client struct {
	logger *logrus.Entry
	glc    *gl.Client
	token  string
	base   string
	dry    bool
}

// NewClient returns an instance of the gitlab client
func NewClient(token, base string) (*Client, error) {
	c := gl.NewClient(nil, token)
	c.SetBaseURL(base)
	return &Client{
		logger: logrus.WithField("client", "gitlab"),
		glc:    c,
		dry:    false,
		token:  token,
		base:   base,
	}, nil
}

// MergeCommentEvent is a wrapper to gitlab type.
type MergeCommentEvent struct {
	*gl.MergeCommentEvent
}

// AddLabel adds a label
func (c *Client) AddLabel(projectID, requestID int, labels []string) error {
	opts := &gl.UpdateMergeRequestOptions{
		Labels: labels,
	}
	_, _, err := c.glc.MergeRequests.UpdateMergeRequest(projectID, requestID, opts)
	if err != nil {
		return err
	}

	return nil
}

// GetMergeRequestLabels returns the list of labels added to the Merge Request.
func (c *Client) GetMergeRequestLabels(projectID, requestID int) ([]string, error) {
	mr, _, err := c.glc.MergeRequests.GetMergeRequest(projectID, requestID)
	if err != nil {
		return nil, err
	}

	return mr.Labels, nil
}
