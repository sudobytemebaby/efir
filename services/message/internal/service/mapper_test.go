package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/sudobytemebaby/efir/services/message/internal/repository"
)

func ptr[T any](v T) *T { return &v }

func TestToContent(t *testing.T) {
	tests := []struct {
		name  string
		input repository.MessageContent
		want  MessageContent
	}{
		{"nil", nil, nil},
		{"text", repository.TextContent{Text: "hi"}, TextContent{Text: "hi"}},
		{"media", repository.MediaContent{
			FileID: "f1", MimeType: "image/png", FileSize: 100, Width: 800, Height: 600,
			ThumbnailID: ptr("thumb"), DurationSec: ptr(int32(10)),
		}, MediaContent{
			FileID: "f1", MimeType: "image/png", FileSize: 100, Width: 800, Height: 600,
			ThumbnailID: ptr("thumb"), DurationSec: ptr(int32(10)),
		}},
		{"file", repository.FileContent{
			FileID: "f2", MimeType: "application/pdf", FileSize: 200, FileName: "doc.pdf",
			DurationSec: ptr(int32(5)),
		}, FileContent{
			FileID: "f2", MimeType: "application/pdf", FileSize: 200, FileName: "doc.pdf",
			DurationSec: ptr(int32(5)),
		}},
		{"voice", repository.VoiceContent{
			FileID: "f3", MimeType: "audio/ogg", FileSize: 300, DurationSec: 15, Waveform: []byte{1, 2, 3},
		}, VoiceContent{
			FileID: "f3", MimeType: "audio/ogg", FileSize: 300, DurationSec: 15, Waveform: []byte{1, 2, 3},
		}},
		{"video_note", repository.VideoNoteContent{
			FileID: "f4", MimeType: "video/mp4", FileSize: 400, DurationSec: 20,
			Width: 240, Height: 240, ThumbnailID: ptr("vthumb"),
		}, VideoNoteContent{
			FileID: "f4", MimeType: "video/mp4", FileSize: 400, DurationSec: 20,
			Width: 240, Height: 240, ThumbnailID: ptr("vthumb"),
		}},
		{"sticker", repository.StickerContent{
			FileID: "f5", MimeType: "image/webp", Emoji: ptr("😀"), SetName: ptr("set1"),
		}, StickerContent{
			FileID: "f5", MimeType: "image/webp", Emoji: ptr("😀"), SetName: ptr("set1"),
		}},
		{"event", repository.EventContent{Text: "joined"}, EventContent{Text: "joined"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toContent(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToRepoContent(t *testing.T) {
	tests := []struct {
		name  string
		input MessageContent
		want  repository.MessageContent
	}{
		{"nil", nil, nil},
		{"text", TextContent{Text: "hi"}, repository.TextContent{Text: "hi"}},
		{"media", MediaContent{
			FileID: "f1", MimeType: "image/png", FileSize: 100, Width: 800, Height: 600,
			ThumbnailID: ptr("thumb"), DurationSec: ptr(int32(10)),
		}, repository.MediaContent{
			FileID: "f1", MimeType: "image/png", FileSize: 100, Width: 800, Height: 600,
			ThumbnailID: ptr("thumb"), DurationSec: ptr(int32(10)),
		}},
		{"file", FileContent{
			FileID: "f2", MimeType: "application/pdf", FileSize: 200, FileName: "doc.pdf",
			DurationSec: ptr(int32(5)),
		}, repository.FileContent{
			FileID: "f2", MimeType: "application/pdf", FileSize: 200, FileName: "doc.pdf",
			DurationSec: ptr(int32(5)),
		}},
		{"voice", VoiceContent{
			FileID: "f3", MimeType: "audio/ogg", FileSize: 300, DurationSec: 15, Waveform: []byte{1, 2, 3},
		}, repository.VoiceContent{
			FileID: "f3", MimeType: "audio/ogg", FileSize: 300, DurationSec: 15, Waveform: []byte{1, 2, 3},
		}},
		{"video_note", VideoNoteContent{
			FileID: "f4", MimeType: "video/mp4", FileSize: 400, DurationSec: 20,
			Width: 240, Height: 240, ThumbnailID: ptr("vthumb"),
		}, repository.VideoNoteContent{
			FileID: "f4", MimeType: "video/mp4", FileSize: 400, DurationSec: 20,
			Width: 240, Height: 240, ThumbnailID: ptr("vthumb"),
		}},
		{"sticker", StickerContent{
			FileID: "f5", MimeType: "image/webp", Emoji: ptr("😀"), SetName: ptr("set1"),
		}, repository.StickerContent{
			FileID: "f5", MimeType: "image/webp", Emoji: ptr("😀"), SetName: ptr("set1"),
		}},
		{"event", EventContent{Text: "joined"}, repository.EventContent{Text: "joined"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toRepoContent(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToPreview(t *testing.T) {
	msgID := uuid.New()
	senderID := uuid.New()
	p := &repository.MessagePreview{
		MessageID:   msgID,
		SenderID:    senderID,
		Type:        repository.MessageTypeText,
		TextPreview: ptr("hello..."),
		FileName:    ptr("file.txt"),
		MimeType:    ptr("text/plain"),
	}

	got := toPreview(p)

	assert.Equal(t, msgID, got.MessageID)
	assert.Equal(t, senderID, got.SenderID)
	assert.Equal(t, MessageType("text"), got.Type)
	assert.Equal(t, ptr("hello..."), got.TextPreview)
	assert.Equal(t, ptr("file.txt"), got.FileName)
	assert.Equal(t, ptr("text/plain"), got.MimeType)
}
