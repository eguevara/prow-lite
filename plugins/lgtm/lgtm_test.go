package lgtm

import (
	"testing"

	"github.com/eguevara/prow-lite/gitlab"
	"github.com/sirupsen/logrus"
)

func Test_handle(t *testing.T) {
	type args struct {
		gc  *gitlab.Client
		log *logrus.Entry
		e   *gitlab.MergeCommentEvent
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := handle(tt.args.gc, tt.args.log, tt.args.e); (err != nil) != tt.wantErr {
				t.Errorf("handle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
