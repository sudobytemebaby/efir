package client

import (
	"context"
	"time"

	messagev1 "github.com/sudobytemebaby/efir/services/shared/gen/message"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Message = messagev1.Message
type MessageType = messagev1.MessageType
type SendMessageRequest = messagev1.SendMessageRequest

const (
	MessageTypeText      MessageType = messagev1.MessageType_MESSAGE_TYPE_TEXT
	MessageTypeImage     MessageType = messagev1.MessageType_MESSAGE_TYPE_IMAGE
	MessageTypeVideo     MessageType = messagev1.MessageType_MESSAGE_TYPE_VIDEO
	MessageTypeVideoNote MessageType = messagev1.MessageType_MESSAGE_TYPE_VIDEO_NOTE
	MessageTypeVoice     MessageType = messagev1.MessageType_MESSAGE_TYPE_VOICE
	MessageTypeAudio     MessageType = messagev1.MessageType_MESSAGE_TYPE_AUDIO
	MessageTypeFile      MessageType = messagev1.MessageType_MESSAGE_TYPE_FILE
	MessageTypeSticker   MessageType = messagev1.MessageType_MESSAGE_TYPE_STICKER
)

type MessageClient struct {
	client  messagev1.MessageServiceClient
	timeout time.Duration
	conn    *grpc.ClientConn
}

func NewMessageClient(addr string, timeout time.Duration) (*MessageClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &MessageClient{
		client:  messagev1.NewMessageServiceClient(conn),
		timeout: timeout,
		conn:    conn,
	}, nil
}

func (c *MessageClient) Close() error {
	return c.conn.Close()
}

func (c *MessageClient) SendMessage(ctx context.Context, req *messagev1.SendMessageRequest) (*messagev1.SendMessageResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.SendMessage(ctx, req)
}

func (c *MessageClient) GetMessages(ctx context.Context, roomID, requesterID string, cursor *string, limit int32) (*messagev1.GetMessagesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.GetMessages(ctx, &messagev1.GetMessagesRequest{
		RoomId:      roomID,
		RequesterId: requesterID,
		Cursor:      cursor,
		Limit:       limit,
	})
}
