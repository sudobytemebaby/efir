package handler_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/sudobytemebaby/efir/services/message/internal/handler"
	"github.com/sudobytemebaby/efir/services/message/internal/service"
	svcmocks "github.com/sudobytemebaby/efir/services/message/internal/service/mocks"
	messagev1 "github.com/sudobytemebaby/efir/services/shared/gen/message"
)

const bufSize = 1024 * 1024

func newGRPCClient(t *testing.T, svc service.MessageService) messagev1.MessageServiceClient {
	t.Helper()

	msgHandler, err := handler.NewMessageHandler(svc)
	require.NoError(t, err)

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	messagev1.RegisterMessageServiceServer(srv, msgHandler)
	t.Cleanup(func() {
		srv.GracefulStop()
		_ = lis.Close()
	})

	go func() {
		if err := srv.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			t.Logf("grpc server error: %v", err)
		}
	}()

	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return messagev1.NewMessageServiceClient(conn)
}

func repoMsg(msgID, roomID, senderID uuid.UUID) *service.Message {
	now := time.Now()
	return &service.Message{
		ID:        msgID,
		RoomID:    roomID,
		SenderID:  senderID,
		Type:      service.MessageTypeText,
		Content:   service.TextContent{Text: "hello"},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestGRPC_SendMessage_Success(t *testing.T) {
	t.Parallel()
	roomID, senderID, msgID := uuid.New(), uuid.New(), uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("SendMessage", mock.Anything, mock.Anything).
		Return(repoMsg(msgID, roomID, senderID), nil)

	client := newGRPCClient(t, svcMock)
	resp, err := client.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   roomID.String(),
		SenderId: senderID.String(),
		Type:     messagev1.MessageType_MESSAGE_TYPE_TEXT,
		Content:  &messagev1.SendMessageRequest_Text{Text: &messagev1.SendTextContent{Text: "hello"}},
	})

	require.NoError(t, err)
	assert.Equal(t, msgID.String(), resp.Message.MessageId)
}

func TestGRPC_SendMessage_NotMember(t *testing.T) {
	t.Parallel()
	roomID, senderID := uuid.New(), uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("SendMessage", mock.Anything, mock.Anything).
		Return(nil, service.ErrNotMember)

	client := newGRPCClient(t, svcMock)
	_, err := client.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   roomID.String(),
		SenderId: senderID.String(),
		Type:     messagev1.MessageType_MESSAGE_TYPE_TEXT,
		Content:  &messagev1.SendMessageRequest_Text{Text: &messagev1.SendTextContent{Text: "hi"}},
	})

	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestGRPC_SendMessage_InvalidArgument(t *testing.T) {
	t.Parallel()
	svcMock := svcmocks.NewMessageService(t)
	client := newGRPCClient(t, svcMock)

	// Missing room_id → proto validation fails before hitting the service.
	_, err := client.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   "",
		SenderId: uuid.New().String(),
	})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPC_GetMessages_Success(t *testing.T) {
	t.Parallel()
	roomID, requesterID := uuid.New(), uuid.New()
	msgID := uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("GetMessages", mock.Anything, roomID, requesterID, (*uuid.UUID)(nil), 10).
		Return([]*service.Message{repoMsg(msgID, roomID, requesterID)}, (*uuid.UUID)(nil), nil)

	client := newGRPCClient(t, svcMock)
	resp, err := client.GetMessages(context.Background(), &messagev1.GetMessagesRequest{
		RoomId:      roomID.String(),
		RequesterId: requesterID.String(),
		Limit:       10,
	})

	require.NoError(t, err)
	require.Len(t, resp.Messages, 1)
	assert.Equal(t, msgID.String(), resp.Messages[0].MessageId)
	assert.Nil(t, resp.NextCursor)
}

func TestGRPC_GetMessages_NotMember(t *testing.T) {
	t.Parallel()
	roomID, requesterID := uuid.New(), uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("GetMessages", mock.Anything, roomID, requesterID, (*uuid.UUID)(nil), 10).
		Return(nil, (*uuid.UUID)(nil), service.ErrNotMember)

	client := newGRPCClient(t, svcMock)
	_, err := client.GetMessages(context.Background(), &messagev1.GetMessagesRequest{
		RoomId:      roomID.String(),
		RequesterId: requesterID.String(),
		Limit:       10,
	})

	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestGRPC_DeleteMessage_Success(t *testing.T) {
	t.Parallel()
	msgID, requesterID := uuid.New(), uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("DeleteMessage", mock.Anything, msgID, requesterID).Return(nil)

	client := newGRPCClient(t, svcMock)
	_, err := client.DeleteMessage(context.Background(), &messagev1.DeleteMessageRequest{
		MessageId:   msgID.String(),
		RequesterId: requesterID.String(),
	})

	require.NoError(t, err)
}

func TestGRPC_DeleteMessage_NotOwner(t *testing.T) {
	t.Parallel()
	msgID, requesterID := uuid.New(), uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("DeleteMessage", mock.Anything, msgID, requesterID).Return(service.ErrNotOwner)

	client := newGRPCClient(t, svcMock)
	_, err := client.DeleteMessage(context.Background(), &messagev1.DeleteMessageRequest{
		MessageId:   msgID.String(),
		RequesterId: requesterID.String(),
	})

	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestGRPC_DeleteMessage_NotFound(t *testing.T) {
	t.Parallel()
	msgID, requesterID := uuid.New(), uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("DeleteMessage", mock.Anything, msgID, requesterID).Return(service.ErrMessageNotFound)

	client := newGRPCClient(t, svcMock)
	_, err := client.DeleteMessage(context.Background(), &messagev1.DeleteMessageRequest{
		MessageId:   msgID.String(),
		RequesterId: requesterID.String(),
	})

	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGRPC_GetMessageById_Success(t *testing.T) {
	t.Parallel()
	msgID, roomID, requesterID := uuid.New(), uuid.New(), uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("GetMessageByID", mock.Anything, msgID, requesterID).
		Return(repoMsg(msgID, roomID, requesterID), nil)

	client := newGRPCClient(t, svcMock)
	resp, err := client.GetMessageById(context.Background(), &messagev1.GetMessageByIdRequest{
		MessageId:   msgID.String(),
		RequesterId: requesterID.String(),
	})

	require.NoError(t, err)
	assert.Equal(t, msgID.String(), resp.Message.MessageId)
}

func TestGRPC_GetMessageById_NotFound(t *testing.T) {
	t.Parallel()
	msgID, requesterID := uuid.New(), uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("GetMessageByID", mock.Anything, msgID, requesterID).
		Return(nil, service.ErrMessageNotFound)

	client := newGRPCClient(t, svcMock)
	_, err := client.GetMessageById(context.Background(), &messagev1.GetMessageByIdRequest{
		MessageId:   msgID.String(),
		RequesterId: requesterID.String(),
	})

	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGRPC_GetMessageById_NotMember(t *testing.T) {
	t.Parallel()
	msgID, requesterID := uuid.New(), uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("GetMessageByID", mock.Anything, msgID, requesterID).
		Return(nil, service.ErrNotMember)

	client := newGRPCClient(t, svcMock)
	_, err := client.GetMessageById(context.Background(), &messagev1.GetMessageByIdRequest{
		MessageId:   msgID.String(),
		RequesterId: requesterID.String(),
	})

	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestGRPC_SendMessage_FileContent(t *testing.T) {
	t.Parallel()
	roomID, senderID, msgID := uuid.New(), uuid.New(), uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("SendMessage", mock.Anything, mock.Anything).
		Return(repoMsg(msgID, roomID, senderID), nil)

	client := newGRPCClient(t, svcMock)
	resp, err := client.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   roomID.String(),
		SenderId: senderID.String(),
		Type:     messagev1.MessageType_MESSAGE_TYPE_FILE,
		Content: &messagev1.SendMessageRequest_File{File: &messagev1.SendFileContent{
			FileId: "f1", MimeType: "application/pdf", FileSize: 100, FileName: "doc.pdf",
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, msgID.String(), resp.Message.MessageId)
}

func TestGRPC_SendMessage_VoiceContent(t *testing.T) {
	t.Parallel()
	roomID, senderID, msgID := uuid.New(), uuid.New(), uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("SendMessage", mock.Anything, mock.Anything).
		Return(repoMsg(msgID, roomID, senderID), nil)

	client := newGRPCClient(t, svcMock)
	resp, err := client.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   roomID.String(),
		SenderId: senderID.String(),
		Type:     messagev1.MessageType_MESSAGE_TYPE_VOICE,
		Content: &messagev1.SendMessageRequest_Voice{Voice: &messagev1.SendVoiceContent{
			FileId: "f1", MimeType: "audio/ogg", FileSize: 300, DurationSec: 15,
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, msgID.String(), resp.Message.MessageId)
}

func TestGRPC_SendMessage_VideoNoteContent(t *testing.T) {
	t.Parallel()
	roomID, senderID, msgID := uuid.New(), uuid.New(), uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("SendMessage", mock.Anything, mock.Anything).
		Return(repoMsg(msgID, roomID, senderID), nil)

	client := newGRPCClient(t, svcMock)
	resp, err := client.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   roomID.String(),
		SenderId: senderID.String(),
		Type:     messagev1.MessageType_MESSAGE_TYPE_VIDEO_NOTE,
		Content: &messagev1.SendMessageRequest_VideoNote{VideoNote: &messagev1.SendVideoNoteContent{
			FileId: "f1", MimeType: "video/mp4", FileSize: 400, DurationSec: 20, Width: 240, Height: 240,
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, msgID.String(), resp.Message.MessageId)
}

func TestGRPC_SendMessage_StickerContent(t *testing.T) {
	t.Parallel()
	roomID, senderID, msgID := uuid.New(), uuid.New(), uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("SendMessage", mock.Anything, mock.Anything).
		Return(repoMsg(msgID, roomID, senderID), nil)

	client := newGRPCClient(t, svcMock)
	resp, err := client.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   roomID.String(),
		SenderId: senderID.String(),
		Type:     messagev1.MessageType_MESSAGE_TYPE_STICKER,
		Content: &messagev1.SendMessageRequest_Sticker{Sticker: &messagev1.SendStickerContent{
			FileId: "f1", MimeType: "image/webp",
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, msgID.String(), resp.Message.MessageId)
}

func TestGRPC_SendMessage_AudioContent(t *testing.T) {
	t.Parallel()
	roomID, senderID, msgID := uuid.New(), uuid.New(), uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("SendMessage", mock.Anything, mock.Anything).
		Return(repoMsg(msgID, roomID, senderID), nil)

	client := newGRPCClient(t, svcMock)
	resp, err := client.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   roomID.String(),
		SenderId: senderID.String(),
		Type:     messagev1.MessageType_MESSAGE_TYPE_AUDIO,
		Content: &messagev1.SendMessageRequest_Audio{Audio: &messagev1.SendAudioContent{
			FileId: "f1", MimeType: "audio/mp3", FileSize: 500, FileName: "song.mp3",
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, msgID.String(), resp.Message.MessageId)
}

func TestGRPC_SendMessage_WithReplyTo(t *testing.T) {
	t.Parallel()
	roomID, senderID, msgID := uuid.New(), uuid.New(), uuid.New()
	replyToID := uuid.New().String()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("SendMessage", mock.Anything, mock.Anything).
		Return(repoMsg(msgID, roomID, senderID), nil)

	client := newGRPCClient(t, svcMock)
	resp, err := client.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:    roomID.String(),
		SenderId:  senderID.String(),
		Type:      messagev1.MessageType_MESSAGE_TYPE_TEXT,
		Content:   &messagev1.SendMessageRequest_Text{Text: &messagev1.SendTextContent{Text: "reply"}},
		ReplyToId: &replyToID,
	})

	require.NoError(t, err)
	assert.Equal(t, msgID.String(), resp.Message.MessageId)
}

func TestGRPC_SendMessage_InvalidReplyTarget(t *testing.T) {
	t.Parallel()
	roomID, senderID := uuid.New(), uuid.New()
	replyToID := uuid.New().String()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("SendMessage", mock.Anything, mock.Anything).
		Return(nil, service.ErrInvalidReplyTarget)

	client := newGRPCClient(t, svcMock)
	_, err := client.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:    roomID.String(),
		SenderId:  senderID.String(),
		Type:      messagev1.MessageType_MESSAGE_TYPE_TEXT,
		Content:   &messagev1.SendMessageRequest_Text{Text: &messagev1.SendTextContent{Text: "reply"}},
		ReplyToId: &replyToID,
	})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPC_SendMessage_InternalError(t *testing.T) {
	t.Parallel()
	roomID, senderID := uuid.New(), uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("SendMessage", mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	client := newGRPCClient(t, svcMock)
	_, err := client.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   roomID.String(),
		SenderId: senderID.String(),
		Type:     messagev1.MessageType_MESSAGE_TYPE_TEXT,
		Content:  &messagev1.SendMessageRequest_Text{Text: &messagev1.SendTextContent{Text: "hi"}},
	})

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGRPC_SendMessage_TypeMismatch(t *testing.T) {
	t.Parallel()
	svcMock := svcmocks.NewMessageService(t)
	client := newGRPCClient(t, svcMock)

	_, err := client.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   uuid.New().String(),
		SenderId: uuid.New().String(),
		Type:     messagev1.MessageType_MESSAGE_TYPE_IMAGE,
		Content:  &messagev1.SendMessageRequest_Text{Text: &messagev1.SendTextContent{Text: "hi"}},
	})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPC_SendMessage_MediaContent(t *testing.T) {
	t.Parallel()
	roomID, senderID, msgID := uuid.New(), uuid.New(), uuid.New()

	svcMock := svcmocks.NewMessageService(t)
	svcMock.On("SendMessage", mock.Anything, mock.Anything).
		Return(repoMsg(msgID, roomID, senderID), nil)

	client := newGRPCClient(t, svcMock)
	resp, err := client.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   roomID.String(),
		SenderId: senderID.String(),
		Type:     messagev1.MessageType_MESSAGE_TYPE_IMAGE,
		Content: &messagev1.SendMessageRequest_Media{Media: &messagev1.SendMediaContent{
			FileId: "f1", MimeType: "image/jpeg", FileSize: 100, Width: 800, Height: 600,
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, msgID.String(), resp.Message.MessageId)
}
