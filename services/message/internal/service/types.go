package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/message/internal/repository"
)

type MessageType string

const (
	MessageTypeText      MessageType = "text"
	MessageTypeImage     MessageType = "image"
	MessageTypeVideo     MessageType = "video"
	MessageTypeVideoNote MessageType = "video_note"
	MessageTypeVoice     MessageType = "voice"
	MessageTypeAudio     MessageType = "audio"
	MessageTypeFile      MessageType = "file"
	MessageTypeSticker   MessageType = "sticker"
	MessageTypeEvent     MessageType = "event"
)

type MessageContent interface {
	messageContent()
}

type TextContent struct {
	Text string
}

func (c TextContent) messageContent() {}

type MediaContent struct {
	FileID      string
	MimeType    string
	FileSize    int64
	Width       int32
	Height      int32
	ThumbnailID *string
	DurationSec *int32
}

func (c MediaContent) messageContent() {}

type FileContent struct {
	FileID      string
	MimeType    string
	FileSize    int64
	FileName    string
	DurationSec *int32
}

func (c FileContent) messageContent() {}

type VoiceContent struct {
	FileID      string
	MimeType    string
	FileSize    int64
	DurationSec int32
	Waveform    []byte
}

func (c VoiceContent) messageContent() {}

type VideoNoteContent struct {
	FileID      string
	MimeType    string
	FileSize    int64
	DurationSec int32
	Width       int32
	Height      int32
	ThumbnailID *string
}

func (c VideoNoteContent) messageContent() {}

type StickerContent struct {
	FileID   string
	MimeType string
	Emoji    *string
	SetName  *string
}

func (c StickerContent) messageContent() {}

type EventContent struct {
	Text string
}

func (c EventContent) messageContent() {}

type MessagePreview struct {
	MessageID   uuid.UUID
	SenderID    uuid.UUID
	Type        MessageType
	TextPreview *string
	FileName    *string
	MimeType    *string
}

type Message struct {
	ID        uuid.UUID
	RoomID    uuid.UUID
	SenderID  uuid.UUID
	Type      MessageType
	Content   MessageContent
	ReplyTo   *MessagePreview
	DeletedAt *time.Time
	EditedAt  *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SendMessageInput struct {
	RoomID    uuid.UUID
	SenderID  uuid.UUID
	Type      MessageType
	Content   MessageContent
	ReplyToID *uuid.UUID
}

func toMessage(m *repository.Message) *Message {
	msg := &Message{
		ID:        m.ID,
		RoomID:    m.RoomID,
		SenderID:  m.SenderID,
		Type:      MessageType(m.Type),
		Content:   toContent(m.Content),
		DeletedAt: m.DeletedAt,
		EditedAt:  m.EditedAt,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
	if m.ReplyTo != nil {
		msg.ReplyTo = toPreview(m.ReplyTo)
	}
	return msg
}

func toPreview(p *repository.MessagePreview) *MessagePreview {
	return &MessagePreview{
		MessageID:   p.MessageID,
		SenderID:    p.SenderID,
		Type:        MessageType(p.Type),
		TextPreview: p.TextPreview,
		FileName:    p.FileName,
		MimeType:    p.MimeType,
	}
}

func toContent(c repository.MessageContent) MessageContent {
	if c == nil {
		return nil
	}
	switch v := c.(type) {
	case repository.TextContent:
		return TextContent{Text: v.Text}
	case repository.MediaContent:
		return MediaContent{
			FileID:      v.FileID,
			MimeType:    v.MimeType,
			FileSize:    v.FileSize,
			Width:       v.Width,
			Height:      v.Height,
			ThumbnailID: v.ThumbnailID,
			DurationSec: v.DurationSec,
		}
	case repository.FileContent:
		return FileContent{
			FileID:      v.FileID,
			MimeType:    v.MimeType,
			FileSize:    v.FileSize,
			FileName:    v.FileName,
			DurationSec: v.DurationSec,
		}
	case repository.VoiceContent:
		return VoiceContent{
			FileID:      v.FileID,
			MimeType:    v.MimeType,
			FileSize:    v.FileSize,
			DurationSec: v.DurationSec,
			Waveform:    v.Waveform,
		}
	case repository.VideoNoteContent:
		return VideoNoteContent{
			FileID:      v.FileID,
			MimeType:    v.MimeType,
			FileSize:    v.FileSize,
			DurationSec: v.DurationSec,
			Width:       v.Width,
			Height:      v.Height,
			ThumbnailID: v.ThumbnailID,
		}
	case repository.StickerContent:
		return StickerContent{
			FileID:   v.FileID,
			MimeType: v.MimeType,
			Emoji:    v.Emoji,
			SetName:  v.SetName,
		}
	case repository.EventContent:
		return EventContent{Text: v.Text}
	default:
		return nil
	}
}

func toRepoContent(c MessageContent) repository.MessageContent {
	if c == nil {
		return nil
	}
	switch v := c.(type) {
	case TextContent:
		return repository.TextContent{Text: v.Text}
	case MediaContent:
		return repository.MediaContent{
			FileID:      v.FileID,
			MimeType:    v.MimeType,
			FileSize:    v.FileSize,
			Width:       v.Width,
			Height:      v.Height,
			ThumbnailID: v.ThumbnailID,
			DurationSec: v.DurationSec,
		}
	case FileContent:
		return repository.FileContent{
			FileID:      v.FileID,
			MimeType:    v.MimeType,
			FileSize:    v.FileSize,
			FileName:    v.FileName,
			DurationSec: v.DurationSec,
		}
	case VoiceContent:
		return repository.VoiceContent{
			FileID:      v.FileID,
			MimeType:    v.MimeType,
			FileSize:    v.FileSize,
			DurationSec: v.DurationSec,
			Waveform:    v.Waveform,
		}
	case VideoNoteContent:
		return repository.VideoNoteContent{
			FileID:      v.FileID,
			MimeType:    v.MimeType,
			FileSize:    v.FileSize,
			DurationSec: v.DurationSec,
			Width:       v.Width,
			Height:      v.Height,
			ThumbnailID: v.ThumbnailID,
		}
	case StickerContent:
		return repository.StickerContent{
			FileID:   v.FileID,
			MimeType: v.MimeType,
			Emoji:    v.Emoji,
			SetName:  v.SetName,
		}
	case EventContent:
		return repository.EventContent{Text: v.Text}
	default:
		return nil
	}
}
