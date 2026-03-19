package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshalTextContent(t *testing.T) {
	original := TextContent{Text: "Hello, world!"}

	data, err := marshalContent(original)
	require.NoError(t, err)

	unmarshaled, err := unmarshalContent(MessageTypeText, data)
	require.NoError(t, err)

	result, ok := unmarshaled.(TextContent)
	require.True(t, ok)
	assert.Equal(t, original.Text, result.Text)
}

func TestMarshalUnmarshalMediaContent(t *testing.T) {
	thumbnail := "thumb_123"
	duration := int32(120)
	original := MediaContent{
		FileID:      "media_456",
		MimeType:    "image/jpeg",
		FileSize:    1024000,
		Width:       1920,
		Height:      1080,
		ThumbnailID: &thumbnail,
		DurationSec: &duration,
	}

	data, err := marshalContent(original)
	require.NoError(t, err)

	unmarshaled, err := unmarshalContent(MessageTypeImage, data)
	require.NoError(t, err)

	result, ok := unmarshaled.(MediaContent)
	require.True(t, ok)
	assert.Equal(t, original.FileID, result.FileID)
	assert.Equal(t, original.MimeType, result.MimeType)
	assert.Equal(t, original.FileSize, result.FileSize)
	assert.Equal(t, original.Width, result.Width)
	assert.Equal(t, original.Height, result.Height)
	assert.Equal(t, *original.ThumbnailID, *result.ThumbnailID)
	assert.Equal(t, *original.DurationSec, *result.DurationSec)
}

func TestMarshalUnmarshalFileContent(t *testing.T) {
	duration := int32(300)
	original := FileContent{
		FileID:      "file_789",
		MimeType:    "application/pdf",
		FileSize:    2048000,
		FileName:    "document.pdf",
		DurationSec: &duration,
	}

	data, err := marshalContent(original)
	require.NoError(t, err)

	unmarshaled, err := unmarshalContent(MessageTypeFile, data)
	require.NoError(t, err)

	result, ok := unmarshaled.(FileContent)
	require.True(t, ok)
	assert.Equal(t, original.FileID, result.FileID)
	assert.Equal(t, original.MimeType, result.MimeType)
	assert.Equal(t, original.FileSize, result.FileSize)
	assert.Equal(t, original.FileName, result.FileName)
	assert.Equal(t, *original.DurationSec, *result.DurationSec)
}

func TestMarshalUnmarshalVoiceContent(t *testing.T) {
	waveform := []byte{1, 2, 3, 4, 5}
	original := VoiceContent{
		FileID:      "voice_111",
		MimeType:    "audio/ogg",
		FileSize:    512000,
		DurationSec: 45,
		Waveform:    waveform,
	}

	data, err := marshalContent(original)
	require.NoError(t, err)

	unmarshaled, err := unmarshalContent(MessageTypeVoice, data)
	require.NoError(t, err)

	result, ok := unmarshaled.(VoiceContent)
	require.True(t, ok)
	assert.Equal(t, original.FileID, result.FileID)
	assert.Equal(t, original.MimeType, result.MimeType)
	assert.Equal(t, original.FileSize, result.FileSize)
	assert.Equal(t, original.DurationSec, result.DurationSec)
	assert.Equal(t, original.Waveform, result.Waveform)
}

func TestMarshalUnmarshalVideoNoteContent(t *testing.T) {
	thumbnail := "vn_thumb_222"
	original := VideoNoteContent{
		FileID:      "video_note_333",
		MimeType:    "video/mp4",
		FileSize:    4096000,
		DurationSec: 60,
		Width:       640,
		Height:      640,
		ThumbnailID: &thumbnail,
	}

	data, err := marshalContent(original)
	require.NoError(t, err)

	unmarshaled, err := unmarshalContent(MessageTypeVideoNote, data)
	require.NoError(t, err)

	result, ok := unmarshaled.(VideoNoteContent)
	require.True(t, ok)
	assert.Equal(t, original.FileID, result.FileID)
	assert.Equal(t, original.MimeType, result.MimeType)
	assert.Equal(t, original.FileSize, result.FileSize)
	assert.Equal(t, original.DurationSec, result.DurationSec)
	assert.Equal(t, original.Width, result.Width)
	assert.Equal(t, original.Height, result.Height)
	assert.Equal(t, *original.ThumbnailID, *result.ThumbnailID)
}

func TestMarshalUnmarshalStickerContent(t *testing.T) {
	emoji := "😀"
	setName := "emoji_set"
	original := StickerContent{
		FileID:   "sticker_444",
		MimeType: "image/webp",
		Emoji:    &emoji,
		SetName:  &setName,
	}

	data, err := marshalContent(original)
	require.NoError(t, err)

	unmarshaled, err := unmarshalContent(MessageTypeSticker, data)
	require.NoError(t, err)

	result, ok := unmarshaled.(StickerContent)
	require.True(t, ok)
	assert.Equal(t, original.FileID, result.FileID)
	assert.Equal(t, original.MimeType, result.MimeType)
	assert.Equal(t, *original.Emoji, *result.Emoji)
	assert.Equal(t, *original.SetName, *result.SetName)
}

func TestMarshalUnmarshalEventContent(t *testing.T) {
	original := EventContent{Text: "User joined the room"}

	data, err := marshalContent(original)
	require.NoError(t, err)

	unmarshaled, err := unmarshalContent(MessageTypeEvent, data)
	require.NoError(t, err)

	result, ok := unmarshaled.(EventContent)
	require.True(t, ok)
	assert.Equal(t, original.Text, result.Text)
}

func TestMarshalUnmarshalAudioContent(t *testing.T) {
	duration := int32(180)
	original := FileContent{
		FileID:      "audio_555",
		MimeType:    "audio/mpeg",
		FileSize:    3072000,
		FileName:    "song.mp3",
		DurationSec: &duration,
	}

	data, err := marshalContent(original)
	require.NoError(t, err)

	unmarshaled, err := unmarshalContent(MessageTypeAudio, data)
	require.NoError(t, err)

	result, ok := unmarshaled.(FileContent)
	require.True(t, ok)
	assert.Equal(t, original.FileID, result.FileID)
	assert.Equal(t, original.MimeType, result.MimeType)
	assert.Equal(t, original.FileSize, result.FileSize)
	assert.Equal(t, original.FileName, result.FileName)
	assert.Equal(t, *original.DurationSec, *result.DurationSec)
}

func TestMarshalUnknownContentType(t *testing.T) {
	data, err := marshalContent(nil)
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestUnmarshalUnknownMessageType(t *testing.T) {
	data := []byte(`{"text": "test"}`)
	result, err := unmarshalContent(MessageType("unknown"), data)
	assert.Error(t, err)
	assert.Nil(t, result)
}
