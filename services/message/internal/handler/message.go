package handler

import (
	"context"
	stderrors "errors"

	"buf.build/go/protovalidate"
	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/message/internal/service"
	messagev1 "github.com/sudobytemebaby/efir/services/shared/gen/message"
	sharederrors "github.com/sudobytemebaby/efir/services/shared/pkg/errors"
	"google.golang.org/protobuf/proto"
)

type messageHandler struct {
	messagev1.UnimplementedMessageServiceServer
	svc       service.MessageService
	validator protovalidate.Validator
}

func NewMessageHandler(svc service.MessageService) (messagev1.MessageServiceServer, error) {
	v, err := protovalidate.New()
	if err != nil {
		return nil, err
	}
	return &messageHandler{
		svc:       svc,
		validator: v,
	}, nil
}

func (h *messageHandler) validate(msg proto.Message) error {
	if err := h.validator.Validate(msg); err != nil {
		return sharederrors.CodeInvalidArgument.Error(err.Error())
	}
	return nil
}

func (h *messageHandler) SendMessage(ctx context.Context, req *messagev1.SendMessageRequest) (*messagev1.SendMessageResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
	}

	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid room_id")
	}

	senderID, err := uuid.Parse(req.SenderId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid sender_id")
	}

	var replyToID *uuid.UUID
	if req.ReplyToId != nil && *req.ReplyToId != "" {
		id, err := uuid.Parse(*req.ReplyToId)
		if err != nil {
			return nil, sharederrors.CodeInvalidArgument.Error("invalid reply_to_id")
		}
		replyToID = &id
	}

	var msgType service.MessageType
	var content service.MessageContent

	switch c := req.Content.(type) {
	case *messagev1.SendMessageRequest_Text:
		if req.Type != messagev1.MessageType_MESSAGE_TYPE_TEXT {
			return nil, sharederrors.CodeInvalidArgument.Error("type must be TEXT for text content")
		}
		msgType = service.MessageTypeText
		content = service.TextContent{Text: c.Text.Text}
	case *messagev1.SendMessageRequest_Media:
		media := c.Media
		var ok bool
		msgType, ok = protoToMessageType(req.Type)
		if !ok || (msgType != service.MessageTypeImage && msgType != service.MessageTypeVideo) {
			return nil, sharederrors.CodeInvalidArgument.Error("type must be IMAGE or VIDEO for media content")
		}
		cc := service.MediaContent{
			FileID:   media.FileId,
			MimeType: media.MimeType,
			FileSize: media.FileSize,
			Width:    media.Width,
			Height:   media.Height,
		}
		if media.ThumbnailId != nil {
			cc.ThumbnailID = media.ThumbnailId
		}
		if media.DurationSec != nil {
			cc.DurationSec = media.DurationSec
		}
		content = cc
	case *messagev1.SendMessageRequest_File:
		if req.Type != messagev1.MessageType_MESSAGE_TYPE_FILE {
			return nil, sharederrors.CodeInvalidArgument.Error("type must be FILE for file content")
		}
		file := c.File
		msgType = service.MessageTypeFile
		cc := service.FileContent{
			FileID:   file.FileId,
			MimeType: file.MimeType,
			FileSize: file.FileSize,
			FileName: file.FileName,
		}
		if file.DurationSec != nil {
			cc.DurationSec = file.DurationSec
		}
		content = cc
	case *messagev1.SendMessageRequest_Voice:
		if req.Type != messagev1.MessageType_MESSAGE_TYPE_VOICE {
			return nil, sharederrors.CodeInvalidArgument.Error("type must be VOICE for voice content")
		}
		voice := c.Voice
		msgType = service.MessageTypeVoice
		cc := service.VoiceContent{
			FileID:      voice.FileId,
			MimeType:    voice.MimeType,
			FileSize:    voice.FileSize,
			DurationSec: voice.DurationSec,
		}
		if voice.Waveform != nil {
			cc.Waveform = voice.Waveform
		}
		content = cc
	case *messagev1.SendMessageRequest_VideoNote:
		if req.Type != messagev1.MessageType_MESSAGE_TYPE_VIDEO_NOTE {
			return nil, sharederrors.CodeInvalidArgument.Error("type must be VIDEO_NOTE for video note content")
		}
		vn := c.VideoNote
		msgType = service.MessageTypeVideoNote
		cc := service.VideoNoteContent{
			FileID:      vn.FileId,
			MimeType:    vn.MimeType,
			FileSize:    vn.FileSize,
			DurationSec: vn.DurationSec,
			Width:       vn.Width,
			Height:      vn.Height,
		}
		if vn.ThumbnailId != nil {
			cc.ThumbnailID = vn.ThumbnailId
		}
		content = cc
	case *messagev1.SendMessageRequest_Sticker:
		if req.Type != messagev1.MessageType_MESSAGE_TYPE_STICKER {
			return nil, sharederrors.CodeInvalidArgument.Error("type must be STICKER for sticker content")
		}
		sticker := c.Sticker
		msgType = service.MessageTypeSticker
		cc := service.StickerContent{
			FileID:   sticker.FileId,
			MimeType: sticker.MimeType,
		}
		if sticker.Emoji != nil {
			cc.Emoji = sticker.Emoji
		}
		if sticker.SetName != nil {
			cc.SetName = sticker.SetName
		}
		content = cc
	case *messagev1.SendMessageRequest_Audio:
		if req.Type != messagev1.MessageType_MESSAGE_TYPE_AUDIO {
			return nil, sharederrors.CodeInvalidArgument.Error("type must be AUDIO for audio content")
		}
		audio := c.Audio
		msgType = service.MessageTypeAudio
		cc := service.FileContent{
			FileID:   audio.FileId,
			MimeType: audio.MimeType,
			FileSize: audio.FileSize,
			FileName: audio.FileName,
		}
		if audio.DurationSec != nil {
			cc.DurationSec = audio.DurationSec
		}
		content = cc
	default:
		return nil, sharederrors.CodeInvalidArgument.Error("empty content")
	}

	input := &service.SendMessageInput{
		RoomID:    roomID,
		SenderID:  senderID,
		Type:      msgType,
		Content:   content,
		ReplyToID: replyToID,
	}

	msg, err := h.svc.SendMessage(ctx, input)
	if err != nil {
		if stderrors.Is(err, service.ErrNotMember) {
			return nil, sharederrors.CodePermissionDenied.Error("must be a room member")
		}
		if stderrors.Is(err, service.ErrInvalidReplyTarget) {
			return nil, sharederrors.CodeInvalidArgument.Error("reply target not found or belongs to a different room")
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &messagev1.SendMessageResponse{
		Message: messageToProto(msg),
	}, nil
}

func (h *messageHandler) GetMessages(ctx context.Context, req *messagev1.GetMessagesRequest) (*messagev1.GetMessagesResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
	}

	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid room_id")
	}

	requesterID, err := uuid.Parse(req.RequesterId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid requester_id")
	}

	var cursor *uuid.UUID
	if req.Cursor != nil && *req.Cursor != "" {
		id, err := uuid.Parse(*req.Cursor)
		if err != nil {
			return nil, sharederrors.CodeInvalidArgument.Error("invalid cursor")
		}
		cursor = &id
	}

	messages, nextCursor, err := h.svc.GetMessages(ctx, roomID, requesterID, cursor, int(req.Limit))
	if err != nil {
		if stderrors.Is(err, service.ErrNotMember) {
			return nil, sharederrors.CodePermissionDenied.Error("must be a room member")
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	protoMessages := make([]*messagev1.Message, len(messages))
	for i, msg := range messages {
		protoMessages[i] = messageToProto(msg)
	}

	var nextCursorStr *string
	if nextCursor != nil {
		s := nextCursor.String()
		nextCursorStr = &s
	}

	return &messagev1.GetMessagesResponse{
		Messages:   protoMessages,
		NextCursor: nextCursorStr,
	}, nil
}

func (h *messageHandler) GetMessageById(ctx context.Context, req *messagev1.GetMessageByIdRequest) (*messagev1.GetMessageByIdResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
	}

	messageID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid message_id")
	}

	requesterID, err := uuid.Parse(req.RequesterId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid requester_id")
	}

	msg, err := h.svc.GetMessageByID(ctx, messageID, requesterID)
	if err != nil {
		if stderrors.Is(err, service.ErrMessageNotFound) {
			return nil, sharederrors.CodeNotFound.Error("message not found")
		}
		if stderrors.Is(err, service.ErrNotMember) {
			return nil, sharederrors.CodePermissionDenied.Error("must be a room member")
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &messagev1.GetMessageByIdResponse{
		Message: messageToProto(msg),
	}, nil
}

func (h *messageHandler) DeleteMessage(ctx context.Context, req *messagev1.DeleteMessageRequest) (*messagev1.DeleteMessageResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
	}

	messageID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid message_id")
	}

	requesterID, err := uuid.Parse(req.RequesterId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid requester_id")
	}

	err = h.svc.DeleteMessage(ctx, messageID, requesterID)
	if err != nil {
		if stderrors.Is(err, service.ErrMessageNotFound) {
			return nil, sharederrors.CodeNotFound.Error("message not found")
		}
		if stderrors.Is(err, service.ErrNotOwner) {
			return nil, sharederrors.CodePermissionDenied.Error("only sender can delete message")
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &messagev1.DeleteMessageResponse{}, nil
}
