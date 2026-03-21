package handler

import (
	"google.golang.org/protobuf/types/known/timestamppb"
)

func timestampToString(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().Format("2006-01-02T15:04:05Z07:00")
}
