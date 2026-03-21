package handler

import (
	"github.com/golang/protobuf/ptypes/timestamp"
)

func timestampToString(ts *timestamp.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().Format("2006-01-02T15:04:05Z07:00")
}
