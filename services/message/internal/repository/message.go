package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type MessageType string

const (
	MessageTypeUnspecified  MessageType = "text"
	MessageTypeText         MessageType = "text"
	MessageTypeImage        MessageType = "image"
	MessageTypeVideo        MessageType = "video"
	MessageTypeVideoNote    MessageType = "video_note"
	MessageTypeVoice        MessageType = "voice"
	MessageTypeAudio        MessageType = "audio"
	MessageTypeFile         MessageType = "file"
	MessageTypeSticker      MessageType = "sticker"
	MessageTypeVideoSticker MessageType = "video_sticker"
	MessageTypeEvent        MessageType = "event"
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

type CreateMessageInput struct {
	RoomID    uuid.UUID
	SenderID  uuid.UUID
	Type      MessageType
	Content   MessageContent
	ReplyToID *uuid.UUID
}

type MessageRepository interface {
	CreateMessage(ctx context.Context, input *CreateMessageInput) (*Message, error)
	GetMessagesByRoomID(ctx context.Context, roomID uuid.UUID, cursor *uuid.UUID, limit int) ([]*Message, *uuid.UUID, error)
	GetMessageByID(ctx context.Context, messageID uuid.UUID) (*Message, error)
	SoftDeleteMessage(ctx context.Context, messageID uuid.UUID) error
}
