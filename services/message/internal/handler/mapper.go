package handler

import (
	"time"

	"github.com/sudobytemebaby/efir/services/message/internal/service"
	messagev1 "github.com/sudobytemebaby/efir/services/shared/gen/message"
	"github.com/sudobytemebaby/efir/services/shared/pkg/mapper"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var messageTypeToProto = map[service.MessageType]messagev1.MessageType{
	service.MessageTypeText:      messagev1.MessageType_MESSAGE_TYPE_TEXT,
	service.MessageTypeImage:     messagev1.MessageType_MESSAGE_TYPE_IMAGE,
	service.MessageTypeVideo:     messagev1.MessageType_MESSAGE_TYPE_VIDEO,
	service.MessageTypeVideoNote: messagev1.MessageType_MESSAGE_TYPE_VIDEO_NOTE,
	service.MessageTypeVoice:     messagev1.MessageType_MESSAGE_TYPE_VOICE,
	service.MessageTypeAudio:     messagev1.MessageType_MESSAGE_TYPE_AUDIO,
	service.MessageTypeFile:      messagev1.MessageType_MESSAGE_TYPE_FILE,
	service.MessageTypeSticker:   messagev1.MessageType_MESSAGE_TYPE_STICKER,
}

var protoToMessageTypeMap = map[messagev1.MessageType]service.MessageType{
	messagev1.MessageType_MESSAGE_TYPE_TEXT:       service.MessageTypeText,
	messagev1.MessageType_MESSAGE_TYPE_IMAGE:      service.MessageTypeImage,
	messagev1.MessageType_MESSAGE_TYPE_VIDEO:      service.MessageTypeVideo,
	messagev1.MessageType_MESSAGE_TYPE_VIDEO_NOTE: service.MessageTypeVideoNote,
	messagev1.MessageType_MESSAGE_TYPE_VOICE:      service.MessageTypeVoice,
	messagev1.MessageType_MESSAGE_TYPE_AUDIO:      service.MessageTypeAudio,
	messagev1.MessageType_MESSAGE_TYPE_FILE:       service.MessageTypeFile,
	messagev1.MessageType_MESSAGE_TYPE_STICKER:    service.MessageTypeSticker,
}

func messageToProto(msg *service.Message) *messagev1.Message {
	result := &messagev1.Message{
		MessageId: msg.ID.String(),
		RoomId:    msg.RoomID.String(),
		SenderId:  msg.SenderID.String(),
		Type:      mapper.Enum(messageTypeToProto, msg.Type),
		IsDeleted: msg.DeletedAt != nil,
		CreatedAt: timestampProto(msg.CreatedAt),
		UpdatedAt: timestampProto(msg.UpdatedAt),
	}

	if msg.EditedAt != nil {
		result.EditedAt = timestampProto(*msg.EditedAt)
	}

	if msg.ReplyTo != nil {
		result.ReplyTo = previewToProto(msg.ReplyTo)
	}

	if msg.DeletedAt != nil {
		return result
	}

	switch c := msg.Content.(type) {
	case service.TextContent:
		result.Content = &messagev1.Message_Text{
			Text: &messagev1.TextContent{Text: c.Text},
		}
	case service.MediaContent:
		media := &messagev1.MediaContent{
			FileId:   c.FileID,
			MimeType: c.MimeType,
			FileSize: c.FileSize,
			Width:    c.Width,
			Height:   c.Height,
		}
		if c.ThumbnailID != nil {
			media.ThumbnailId = c.ThumbnailID
		}
		if c.DurationSec != nil {
			media.DurationSec = c.DurationSec
		}
		result.Content = &messagev1.Message_Media{Media: media}
	case service.FileContent:
		file := &messagev1.FileContent{
			FileId:   c.FileID,
			MimeType: c.MimeType,
			FileSize: c.FileSize,
			FileName: c.FileName,
		}
		if c.DurationSec != nil {
			file.DurationSec = c.DurationSec
		}
		result.Content = &messagev1.Message_File{File: file}
	case service.VoiceContent:
		voice := &messagev1.VoiceContent{
			FileId:      c.FileID,
			MimeType:    c.MimeType,
			FileSize:    c.FileSize,
			DurationSec: c.DurationSec,
		}
		if c.Waveform != nil {
			voice.Waveform = c.Waveform
		}
		result.Content = &messagev1.Message_Voice{Voice: voice}
	case service.VideoNoteContent:
		vn := &messagev1.VideoNoteContent{
			FileId:      c.FileID,
			MimeType:    c.MimeType,
			FileSize:    c.FileSize,
			DurationSec: c.DurationSec,
			Width:       c.Width,
			Height:      c.Height,
		}
		if c.ThumbnailID != nil {
			vn.ThumbnailId = c.ThumbnailID
		}
		result.Content = &messagev1.Message_VideoNote{VideoNote: vn}
	case service.StickerContent:
		sticker := &messagev1.StickerContent{
			FileId:   c.FileID,
			MimeType: c.MimeType,
		}
		if c.Emoji != nil {
			sticker.Emoji = c.Emoji
		}
		if c.SetName != nil {
			sticker.SetName = c.SetName
		}
		result.Content = &messagev1.Message_Sticker{Sticker: sticker}
	case service.EventContent:
		result.Content = &messagev1.Message_Event{
			Event: &messagev1.EventContent{Text: c.Text},
		}
	}

	return result
}

func previewToProto(preview *service.MessagePreview) *messagev1.MessagePreview {
	result := &messagev1.MessagePreview{
		MessageId: preview.MessageID.String(),
		SenderId:  preview.SenderID.String(),
		Type:      mapper.Enum(messageTypeToProto, preview.Type),
	}

	if preview.TextPreview != nil {
		result.TextPreview = preview.TextPreview
	}
	if preview.FileName != nil {
		result.FileName = preview.FileName
	}
	if preview.MimeType != nil {
		result.MimeType = preview.MimeType
	}

	return result
}

func protoToMessageType(msgType messagev1.MessageType) (service.MessageType, bool) {
	return mapper.EnumWithOk(protoToMessageTypeMap, msgType)
}

func timestampProto(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}
