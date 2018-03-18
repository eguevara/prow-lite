package api

// All enabled plugins. We need to empty import them like this so that they
// will be linked into any hook binary.
import (
	_ "github.com/eguevara/prow-lite/plugins/lgtm"
)
