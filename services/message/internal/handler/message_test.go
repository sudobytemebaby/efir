package handler

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/sudobytemebaby/efir/services/message/internal/service"
	svcmocks "github.com/sudobytemebaby/efir/services/message/internal/service/mocks"
	messagev1 "github.com/sudobytemebaby/efir/services/shared/gen/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSendMessage_Validation(t *testing.T) {
	h, err := NewMessageHandler(svcmocks.NewMessageService(t))
	require.NoError(t, err)

	_, err = h.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   "",
		SenderId: "user-123",
	})
	assert.Error(t, err)

	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}

func TestSendMessage_EmptyContent(t *testing.T) {
	h, err := NewMessageHandler(svcmocks.NewMessageService(t))
	require.NoError(t, err)

	_, err = h.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   uuid.New().String(),
		SenderId: uuid.New().String(),
	})
	assert.Error(t, err)

	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}

func TestSendMessage_ErrorMapping(t *testing.T) {
	mockSvc := svcmocks.NewMessageService(t)
	mockSvc.On("SendMessage", context.Background(), mock.MatchedBy(func(in *service.SendMessageInput) bool {
		return in.Type == service.MessageTypeText && in.Content == service.TextContent{Text: "Hello"}
	})).Return(nil, service.ErrNotMember)

	h, err := NewMessageHandler(mockSvc)
	require.NoError(t, err)

	_, err = h.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   uuid.New().String(),
		SenderId: uuid.New().String(),
		Type:     messagev1.MessageType_MESSAGE_TYPE_TEXT,
		Content: &messagev1.SendMessageRequest_Text{
			Text: &messagev1.SendTextContent{Text: "Hello"},
		},
	})
	assert.Error(t, err)

	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, s.Code())
}

func TestSendMessage_InvalidReplyTargetMapping(t *testing.T) {
	mockSvc := svcmocks.NewMessageService(t)
	mockSvc.On("SendMessage", context.Background(), mock.Anything).Return(nil, service.ErrInvalidReplyTarget)

	h, err := NewMessageHandler(mockSvc)
	require.NoError(t, err)

	replyToID := uuid.New().String()
	_, err = h.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:    uuid.New().String(),
		SenderId:  uuid.New().String(),
		Type:      messagev1.MessageType_MESSAGE_TYPE_TEXT,
		ReplyToId: &replyToID,
		Content: &messagev1.SendMessageRequest_Text{
			Text: &messagev1.SendTextContent{Text: "Hello"},
		},
	})
	assert.Error(t, err)

	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}

func TestGetMessages_Validation(t *testing.T) {
	h, err := NewMessageHandler(svcmocks.NewMessageService(t))
	require.NoError(t, err)

	_, err = h.GetMessages(context.Background(), &messagev1.GetMessagesRequest{
		RoomId:      uuid.New().String(),
		RequesterId: uuid.New().String(),
		Limit:       0,
	})
	assert.Error(t, err)

	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}

func TestDeleteMessage_ErrorMapping_NotOwner(t *testing.T) {
	mockSvc := svcmocks.NewMessageService(t)
	mockSvc.On("DeleteMessage", context.Background(), mock.Anything, mock.Anything).Return(service.ErrNotOwner)

	h, err := NewMessageHandler(mockSvc)
	require.NoError(t, err)

	_, err = h.DeleteMessage(context.Background(), &messagev1.DeleteMessageRequest{
		MessageId:   uuid.New().String(),
		RequesterId: uuid.New().String(),
	})
	assert.Error(t, err)

	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, s.Code())
}

func TestDeleteMessage_ErrorMapping_NotFound(t *testing.T) {
	mockSvc := svcmocks.NewMessageService(t)
	mockSvc.On("DeleteMessage", context.Background(), mock.Anything, mock.Anything).Return(service.ErrMessageNotFound)

	h, err := NewMessageHandler(mockSvc)
	require.NoError(t, err)

	_, err = h.DeleteMessage(context.Background(), &messagev1.DeleteMessageRequest{
		MessageId:   uuid.New().String(),
		RequesterId: uuid.New().String(),
	})
	assert.Error(t, err)

	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, s.Code())
}

func TestMapMessageToProto_DeletedMessage(t *testing.T) {
	now := time.Now()
	msg := &service.Message{
		ID:        uuid.New(),
		RoomID:    uuid.New(),
		SenderID:  uuid.New(),
		Type:      service.MessageTypeText,
		DeletedAt: &now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	result := messageToProto(msg)
	assert.True(t, result.IsDeleted)
}

func TestMapMessageToProto_WithReplyTo(t *testing.T) {
	now := time.Now()
	replyToID := uuid.New()
	preview := &service.MessagePreview{
		MessageID:   replyToID,
		SenderID:    uuid.New(),
		Type:        service.MessageTypeText,
		TextPreview: strPtr("Original message"),
	}

	msg := &service.Message{
		ID:        uuid.New(),
		RoomID:    uuid.New(),
		SenderID:  uuid.New(),
		Type:      service.MessageTypeText,
		Content:   service.TextContent{Text: "Reply"},
		ReplyTo:   preview,
		CreatedAt: now,
		UpdatedAt: now,
	}

	result := messageToProto(msg)
	assert.NotNil(t, result.ReplyTo)
	assert.Equal(t, replyToID.String(), result.ReplyTo.MessageId)
	assert.NotNil(t, result.ReplyTo.TextPreview)
	assert.Equal(t, "Original message", *result.ReplyTo.TextPreview)
}

func TestMapPreviewToProto(t *testing.T) {
	preview := &service.MessagePreview{
		MessageID:   uuid.New(),
		SenderID:    uuid.New(),
		Type:        service.MessageTypeText,
		TextPreview: strPtr("Preview text"),
		FileName:    strPtr("file.pdf"),
		MimeType:    strPtr("application/pdf"),
	}

	result := previewToProto(preview)
	assert.Equal(t, preview.MessageID.String(), result.MessageId)
	assert.Equal(t, preview.SenderID.String(), result.SenderId)
	assert.NotNil(t, result.TextPreview)
	assert.Equal(t, "Preview text", *result.TextPreview)
}

func strPtr(s string) *string {
	return &s
}

func int32Ptr(v int32) *int32 { return &v }

func TestMessageToProto_AllContentTypes(t *testing.T) {
	now := time.Now()
	base := func(content service.MessageContent, msgType service.MessageType) *service.Message {
		return &service.Message{
			ID: uuid.New(), RoomID: uuid.New(), SenderID: uuid.New(),
			Type: msgType, Content: content, CreatedAt: now, UpdatedAt: now,
		}
	}

	tests := []struct {
		name    string
		msg     *service.Message
		checkFn func(t *testing.T, p *messagev1.Message)
	}{
		{"text", base(service.TextContent{Text: "hi"}, service.MessageTypeText), func(t *testing.T, p *messagev1.Message) {
			require.NotNil(t, p.GetText())
			assert.Equal(t, "hi", p.GetText().Text)
		}},
		{"media", base(service.MediaContent{
			FileID: "f1", MimeType: "image/png", FileSize: 100, Width: 800, Height: 600,
			ThumbnailID: strPtr("thumb"), DurationSec: int32Ptr(10),
		}, service.MessageTypeImage), func(t *testing.T, p *messagev1.Message) {
			require.NotNil(t, p.GetMedia())
			assert.Equal(t, "f1", p.GetMedia().FileId)
			assert.Equal(t, "thumb", *p.GetMedia().ThumbnailId)
			assert.Equal(t, int32(10), *p.GetMedia().DurationSec)
		}},
		{"file", base(service.FileContent{
			FileID: "f2", MimeType: "application/pdf", FileSize: 200, FileName: "doc.pdf",
			DurationSec: int32Ptr(5),
		}, service.MessageTypeFile), func(t *testing.T, p *messagev1.Message) {
			require.NotNil(t, p.GetFile())
			assert.Equal(t, "doc.pdf", p.GetFile().FileName)
			assert.Equal(t, int32(5), *p.GetFile().DurationSec)
		}},
		{"voice", base(service.VoiceContent{
			FileID: "f3", MimeType: "audio/ogg", FileSize: 300, DurationSec: 15, Waveform: []byte{1, 2},
		}, service.MessageTypeVoice), func(t *testing.T, p *messagev1.Message) {
			require.NotNil(t, p.GetVoice())
			assert.Equal(t, int32(15), p.GetVoice().DurationSec)
			assert.Equal(t, []byte{1, 2}, p.GetVoice().Waveform)
		}},
		{"video_note", base(service.VideoNoteContent{
			FileID: "f4", MimeType: "video/mp4", FileSize: 400, DurationSec: 20,
			Width: 240, Height: 240, ThumbnailID: strPtr("vthumb"),
		}, service.MessageTypeVideoNote), func(t *testing.T, p *messagev1.Message) {
			require.NotNil(t, p.GetVideoNote())
			assert.Equal(t, int32(240), p.GetVideoNote().Width)
			assert.Equal(t, "vthumb", *p.GetVideoNote().ThumbnailId)
		}},
		{"sticker", base(service.StickerContent{
			FileID: "f5", MimeType: "image/webp", Emoji: strPtr("😀"), SetName: strPtr("set1"),
		}, service.MessageTypeSticker), func(t *testing.T, p *messagev1.Message) {
			require.NotNil(t, p.GetSticker())
			assert.Equal(t, "😀", *p.GetSticker().Emoji)
			assert.Equal(t, "set1", *p.GetSticker().SetName)
		}},
		{"event", base(service.EventContent{Text: "joined"}, service.MessageTypeEvent), func(t *testing.T, p *messagev1.Message) {
			require.NotNil(t, p.GetEvent())
			assert.Equal(t, "joined", p.GetEvent().Text)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := messageToProto(tt.msg)
			assert.False(t, result.IsDeleted)
			tt.checkFn(t, result)
		})
	}
}

func TestMessageToProto_WithEditedAt(t *testing.T) {
	now := time.Now()
	msg := &service.Message{
		ID: uuid.New(), RoomID: uuid.New(), SenderID: uuid.New(),
		Type: service.MessageTypeText, Content: service.TextContent{Text: "edited"},
		EditedAt: &now, CreatedAt: now, UpdatedAt: now,
	}
	result := messageToProto(msg)
	assert.NotNil(t, result.EditedAt)
}

func TestProtoToMessageType(t *testing.T) {
	tests := []struct {
		proto messagev1.MessageType
		want  service.MessageType
		ok    bool
	}{
		{messagev1.MessageType_MESSAGE_TYPE_TEXT, service.MessageTypeText, true},
		{messagev1.MessageType_MESSAGE_TYPE_IMAGE, service.MessageTypeImage, true},
		{messagev1.MessageType_MESSAGE_TYPE_VIDEO, service.MessageTypeVideo, true},
		{messagev1.MessageType_MESSAGE_TYPE_VIDEO_NOTE, service.MessageTypeVideoNote, true},
		{messagev1.MessageType_MESSAGE_TYPE_VOICE, service.MessageTypeVoice, true},
		{messagev1.MessageType_MESSAGE_TYPE_AUDIO, service.MessageTypeAudio, true},
		{messagev1.MessageType_MESSAGE_TYPE_FILE, service.MessageTypeFile, true},
		{messagev1.MessageType_MESSAGE_TYPE_STICKER, service.MessageTypeSticker, true},
		{messagev1.MessageType_MESSAGE_TYPE_UNSPECIFIED, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.proto.String(), func(t *testing.T) {
			got, ok := protoToMessageType(tt.proto)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
