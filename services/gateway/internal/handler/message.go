package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/sudobytemebaby/efir/services/gateway/internal/client"
	"github.com/sudobytemebaby/efir/services/gateway/internal/middleware"
	messagev1 "github.com/sudobytemebaby/efir/services/shared/gen/message"
	"github.com/sudobytemebaby/efir/services/shared/pkg/errors"
)

type MessageHandler struct {
	messageClient *client.MessageClient
}

func NewMessageHandler(messageClient *client.MessageClient) *MessageHandler {
	return &MessageHandler{
		messageClient: messageClient,
	}
}

func (h *MessageHandler) Register(r chi.Router) {
	r.Post("/rooms/{id}/messages", h.sendMessage)
	r.Get("/rooms/{id}/messages", h.getMessages)
}

type sendMessageRequest struct {
	Type      string                `json:"type"`
	Text      *sendTextContent      `json:"text,omitempty"`
	Media     *sendMediaContent     `json:"media,omitempty"`
	File      *sendFileContent      `json:"file,omitempty"`
	Voice     *sendVoiceContent     `json:"voice,omitempty"`
	VideoNote *sendVideoNoteContent `json:"video_note,omitempty"`
	Sticker   *sendStickerContent   `json:"sticker,omitempty"`
	Audio     *sendAudioContent     `json:"audio,omitempty"`
	ReplyTo   *string               `json:"reply_to,omitempty"`
}

type sendTextContent struct {
	Text string `json:"text"`
}

type sendMediaContent struct {
	FileID      string  `json:"file_id"`
	MimeType    string  `json:"mime_type"`
	FileSize    int64   `json:"file_size"`
	Width       int32   `json:"width"`
	Height      int32   `json:"height"`
	ThumbnailID *string `json:"thumbnail_id,omitempty"`
	DurationSec *int32  `json:"duration_sec,omitempty"`
}

type sendFileContent struct {
	FileID      string `json:"file_id"`
	MimeType    string `json:"mime_type"`
	FileSize    int64  `json:"file_size"`
	FileName    string `json:"file_name"`
	DurationSec *int32 `json:"duration_sec,omitempty"`
}

type sendVoiceContent struct {
	FileID      string `json:"file_id"`
	MimeType    string `json:"mime_type"`
	FileSize    int64  `json:"file_size"`
	DurationSec int32  `json:"duration_sec"`
	Waveform    []byte `json:"waveform,omitempty"`
}

type sendVideoNoteContent struct {
	FileID      string  `json:"file_id"`
	MimeType    string  `json:"mime_type"`
	FileSize    int64   `json:"file_size"`
	DurationSec int32   `json:"duration_sec"`
	Width       int32   `json:"width"`
	Height      int32   `json:"height"`
	ThumbnailID *string `json:"thumbnail_id,omitempty"`
}

type sendStickerContent struct {
	FileID   string  `json:"file_id"`
	MimeType string  `json:"mime_type"`
	Emoji    *string `json:"emoji,omitempty"`
	SetName  *string `json:"set_name,omitempty"`
}

type sendAudioContent struct {
	FileID      string `json:"file_id"`
	MimeType    string `json:"mime_type"`
	FileSize    int64  `json:"file_size"`
	FileName    string `json:"file_name"`
	DurationSec *int32 `json:"duration_sec,omitempty"`
}

func messageTypeFromString(t string) (messagev1.MessageType, bool) {
	switch t {
	case "text":
		return messagev1.MessageType_MESSAGE_TYPE_TEXT, true
	case "image":
		return messagev1.MessageType_MESSAGE_TYPE_IMAGE, true
	case "video":
		return messagev1.MessageType_MESSAGE_TYPE_VIDEO, true
	case "video_note":
		return messagev1.MessageType_MESSAGE_TYPE_VIDEO_NOTE, true
	case "voice":
		return messagev1.MessageType_MESSAGE_TYPE_VOICE, true
	case "audio":
		return messagev1.MessageType_MESSAGE_TYPE_AUDIO, true
	case "file":
		return messagev1.MessageType_MESSAGE_TYPE_FILE, true
	case "sticker":
		return messagev1.MessageType_MESSAGE_TYPE_STICKER, true
	default:
		return 0, false
	}
}

func (h *MessageHandler) sendMessage(w http.ResponseWriter, r *http.Request) {
	senderID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	roomID := chi.URLParam(r, "id")
	if roomID == "" {
		http.Error(w, "missing room id", http.StatusBadRequest)
		return
	}

	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	msgType, ok := messageTypeFromString(req.Type)
	if !ok {
		http.Error(w, "invalid message type", http.StatusBadRequest)
		return
	}

	grpcReq := &messagev1.SendMessageRequest{
		RoomId:    roomID,
		SenderId:  senderID,
		Type:      msgType,
		ReplyToId: req.ReplyTo,
	}

	switch msgType {
	case messagev1.MessageType_MESSAGE_TYPE_TEXT:
		if req.Text == nil {
			http.Error(w, "missing text content", http.StatusBadRequest)
			return
		}
		grpcReq.Content = &messagev1.SendMessageRequest_Text{Text: &messagev1.SendTextContent{Text: req.Text.Text}}
	case messagev1.MessageType_MESSAGE_TYPE_IMAGE, messagev1.MessageType_MESSAGE_TYPE_VIDEO:
		if req.Media == nil {
			http.Error(w, "missing media content", http.StatusBadRequest)
			return
		}
		grpcReq.Content = &messagev1.SendMessageRequest_Media{Media: &messagev1.SendMediaContent{
			FileId: req.Media.FileID, MimeType: req.Media.MimeType, FileSize: req.Media.FileSize,
			Width: req.Media.Width, Height: req.Media.Height, ThumbnailId: req.Media.ThumbnailID, DurationSec: req.Media.DurationSec,
		}}
	case messagev1.MessageType_MESSAGE_TYPE_FILE:
		if req.File == nil {
			http.Error(w, "missing file content", http.StatusBadRequest)
			return
		}
		grpcReq.Content = &messagev1.SendMessageRequest_File{File: &messagev1.SendFileContent{
			FileId: req.File.FileID, MimeType: req.File.MimeType, FileSize: req.File.FileSize,
			FileName: req.File.FileName, DurationSec: req.File.DurationSec,
		}}
	case messagev1.MessageType_MESSAGE_TYPE_VOICE:
		if req.Voice == nil {
			http.Error(w, "missing voice content", http.StatusBadRequest)
			return
		}
		grpcReq.Content = &messagev1.SendMessageRequest_Voice{Voice: &messagev1.SendVoiceContent{
			FileId: req.Voice.FileID, MimeType: req.Voice.MimeType, FileSize: req.Voice.FileSize,
			DurationSec: req.Voice.DurationSec, Waveform: req.Voice.Waveform,
		}}
	case messagev1.MessageType_MESSAGE_TYPE_VIDEO_NOTE:
		if req.VideoNote == nil {
			http.Error(w, "missing video_note content", http.StatusBadRequest)
			return
		}
		grpcReq.Content = &messagev1.SendMessageRequest_VideoNote{VideoNote: &messagev1.SendVideoNoteContent{
			FileId: req.VideoNote.FileID, MimeType: req.VideoNote.MimeType, FileSize: req.VideoNote.FileSize,
			DurationSec: req.VideoNote.DurationSec, Width: req.VideoNote.Width, Height: req.VideoNote.Height,
			ThumbnailId: req.VideoNote.ThumbnailID,
		}}
	case messagev1.MessageType_MESSAGE_TYPE_STICKER:
		if req.Sticker == nil {
			http.Error(w, "missing sticker content", http.StatusBadRequest)
			return
		}
		grpcReq.Content = &messagev1.SendMessageRequest_Sticker{Sticker: &messagev1.SendStickerContent{
			FileId: req.Sticker.FileID, MimeType: req.Sticker.MimeType,
			Emoji: req.Sticker.Emoji, SetName: req.Sticker.SetName,
		}}
	case messagev1.MessageType_MESSAGE_TYPE_AUDIO:
		if req.Audio == nil {
			http.Error(w, "missing audio content", http.StatusBadRequest)
			return
		}
		grpcReq.Content = &messagev1.SendMessageRequest_Audio{Audio: &messagev1.SendAudioContent{
			FileId: req.Audio.FileID, MimeType: req.Audio.MimeType, FileSize: req.Audio.FileSize,
			FileName: req.Audio.FileName, DurationSec: req.Audio.DurationSec,
		}}
	}

	resp, err := h.messageClient.SendMessage(r.Context(), grpcReq)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to send message", "error", err)
		code := errors.FromError(err)
		http.Error(w, err.Error(), code.ToHTTPCode())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(messageToResponse(resp.Message)); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}

type messageResponse struct {
	MessageID string            `json:"message_id"`
	RoomID    string            `json:"room_id"`
	SenderID  string            `json:"sender_id"`
	Type      string            `json:"type"`
	IsDeleted bool              `json:"is_deleted"`
	EditedAt  string            `json:"edited_at,omitempty"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
	ReplyTo   *messagePreview   `json:"reply_to,omitempty"`
	Text      string            `json:"text,omitempty"`
	Media     *mediaPreview     `json:"media,omitempty"`
	File      *filePreview      `json:"file,omitempty"`
	Voice     *voicePreview     `json:"voice,omitempty"`
	VideoNote *videoNotePreview `json:"video_note,omitempty"`
	Sticker   *stickerPreview   `json:"sticker,omitempty"`
	Audio     *audioPreview     `json:"audio,omitempty"`
}

type messagePreview struct {
	MessageID string `json:"message_id"`
	SenderID  string `json:"sender_id"`
	Type      string `json:"type"`
}

type mediaPreview struct {
	FileID    string `json:"file_id"`
	MimeType  string `json:"mime_type"`
	FileSize  int64  `json:"file_size"`
	Width     int32  `json:"width"`
	Height    int32  `json:"height"`
	Thumbnail string `json:"thumbnail,omitempty"`
}

type filePreview struct {
	FileID      string `json:"file_id"`
	MimeType    string `json:"mime_type"`
	FileSize    int64  `json:"file_size"`
	FileName    string `json:"file_name"`
	DurationSec int32  `json:"duration_sec,omitempty"`
}

type voicePreview struct {
	FileID      string `json:"file_id"`
	MimeType    string `json:"mime_type"`
	FileSize    int64  `json:"file_size"`
	DurationSec int32  `json:"duration_sec"`
}

type videoNotePreview struct {
	FileID      string `json:"file_id"`
	MimeType    string `json:"mime_type"`
	FileSize    int64  `json:"file_size"`
	DurationSec int32  `json:"duration_sec"`
	Width       int32  `json:"width"`
	Height      int32  `json:"height"`
	Thumbnail   string `json:"thumbnail,omitempty"`
}

type stickerPreview struct {
	FileID   string `json:"file_id"`
	MimeType string `json:"mime_type"`
	Emoji    string `json:"emoji,omitempty"`
}

type audioPreview struct {
	FileID      string `json:"file_id"`
	MimeType    string `json:"mime_type"`
	FileSize    int64  `json:"file_size"`
	FileName    string `json:"file_name"`
	DurationSec int32  `json:"duration_sec,omitempty"`
}

func messageTypeToString(t messagev1.MessageType) string {
	switch t {
	case messagev1.MessageType_MESSAGE_TYPE_TEXT:
		return "text"
	case messagev1.MessageType_MESSAGE_TYPE_IMAGE:
		return "image"
	case messagev1.MessageType_MESSAGE_TYPE_VIDEO:
		return "video"
	case messagev1.MessageType_MESSAGE_TYPE_VIDEO_NOTE:
		return "video_note"
	case messagev1.MessageType_MESSAGE_TYPE_VOICE:
		return "voice"
	case messagev1.MessageType_MESSAGE_TYPE_AUDIO:
		return "audio"
	case messagev1.MessageType_MESSAGE_TYPE_FILE:
		return "file"
	case messagev1.MessageType_MESSAGE_TYPE_STICKER:
		return "sticker"
	default:
		return "unspecified"
	}
}

func messageToResponse(msg *messagev1.Message) messageResponse {
	resp := messageResponse{
		MessageID: msg.MessageId,
		RoomID:    msg.RoomId,
		SenderID:  msg.SenderId,
		Type:      messageTypeToString(msg.Type),
		IsDeleted: msg.IsDeleted,
		CreatedAt: timestampToString(msg.CreatedAt),
		UpdatedAt: timestampToString(msg.UpdatedAt),
	}
	if msg.EditedAt != nil {
		resp.EditedAt = timestampToString(msg.EditedAt)
	}
	if msg.ReplyTo != nil {
		resp.ReplyTo = &messagePreview{
			MessageID: msg.ReplyTo.MessageId,
			SenderID:  msg.ReplyTo.SenderId,
			Type:      messageTypeToString(msg.ReplyTo.Type),
		}
	}
	if text := msg.GetText(); text != nil {
		resp.Text = text.Text
	}
	if media := msg.GetMedia(); media != nil {
		preview := &mediaPreview{
			FileID: media.FileId, MimeType: media.MimeType, FileSize: media.FileSize,
			Width: media.Width, Height: media.Height,
		}
		if media.ThumbnailId != nil {
			preview.Thumbnail = *media.ThumbnailId
		}
		resp.Media = preview
	}
	if file := msg.GetFile(); file != nil {
		preview := &filePreview{
			FileID: file.FileId, MimeType: file.MimeType, FileSize: file.FileSize,
			FileName: file.FileName,
		}
		if file.DurationSec != nil {
			preview.DurationSec = *file.DurationSec
		}
		resp.File = preview
	}
	if voice := msg.GetVoice(); voice != nil {
		resp.Voice = &voicePreview{
			FileID: voice.FileId, MimeType: voice.MimeType, FileSize: voice.FileSize,
			DurationSec: voice.DurationSec,
		}
	}
	if videoNote := msg.GetVideoNote(); videoNote != nil {
		preview := &videoNotePreview{
			FileID: videoNote.FileId, MimeType: videoNote.MimeType, FileSize: videoNote.FileSize,
			DurationSec: videoNote.DurationSec, Width: videoNote.Width, Height: videoNote.Height,
		}
		if videoNote.ThumbnailId != nil {
			preview.Thumbnail = *videoNote.ThumbnailId
		}
		resp.VideoNote = preview
	}
	if sticker := msg.GetSticker(); sticker != nil {
		preview := &stickerPreview{
			FileID: sticker.FileId, MimeType: sticker.MimeType,
		}
		if sticker.Emoji != nil {
			preview.Emoji = *sticker.Emoji
		}
		resp.Sticker = preview
	}
	return resp
}

func (h *MessageHandler) getMessages(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	roomID := chi.URLParam(r, "id")
	if roomID == "" {
		http.Error(w, "missing room id", http.StatusBadRequest)
		return
	}

	var cursor *string
	if c := r.URL.Query().Get("cursor"); c != "" {
		cursor = &c
	}

	limit := int32(50)
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.ParseInt(l, 10, 32); err == nil {
			limit = int32(parsed)
		}
	}

	resp, err := h.messageClient.GetMessages(r.Context(), roomID, requesterID, cursor, limit)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get messages", "error", err)
		code := errors.FromError(err)
		http.Error(w, err.Error(), code.ToHTTPCode())
		return
	}

	messages := make([]messageResponse, 0, len(resp.Messages))
	for _, msg := range resp.Messages {
		messages = append(messages, messageToResponse(msg))
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Messages   []messageResponse `json:"messages"`
		NextCursor *string           `json:"next_cursor,omitempty"`
	}{
		Messages:   messages,
		NextCursor: resp.NextCursor,
	}); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}
