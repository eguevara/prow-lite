// Package logrusutil implements some helpers for using logrus
package logrusutil

import (
	"github.com/sirupsen/logrus"
)

// DefaultFieldsFormatter wraps another logrus.Formatter, injecting
// DefaultFields into each Format() call, existing fields are preserved
// if they have the same key
type DefaultFieldsFormatter struct {
	WrappedFormatter logrus.Formatter
	DefaultFields    logrus.Fields
}

// NewDefaultFieldsFormatter returns a DefaultFieldsFormatter,
// if wrappedFormatter is nil &logrus.JSONFormatter{} will be used instead
func NewDefaultFieldsFormatter(
	wrappedFormatter logrus.Formatter, defaultFields logrus.Fields,
) *DefaultFieldsFormatter {
	res := &DefaultFieldsFormatter{
		WrappedFormatter: wrappedFormatter,
		DefaultFields:    defaultFields,
	}
	if res.WrappedFormatter == nil {
		res.WrappedFormatter = &logrus.JSONFormatter{}
	}
	return res
}

// Format implements logrus.Formatter's Format
func (d *DefaultFieldsFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	if entry != nil {
		for k, v := range d.DefaultFields {
			if _, exists := entry.Data[k]; !exists {
				entry.Data[k] = v
			}
		}
	}
	return d.WrappedFormatter.Format(entry)
}
