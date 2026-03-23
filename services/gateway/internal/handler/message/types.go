package message

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
