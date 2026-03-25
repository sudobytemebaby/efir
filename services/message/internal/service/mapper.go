package service

import (
	"github.com/sudobytemebaby/efir/services/message/internal/repository"
)

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
