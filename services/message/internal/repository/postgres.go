package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrMessageNotFound = errors.New("message not found")

func marshalContent(content MessageContent) ([]byte, error) {
	switch c := content.(type) {
	case TextContent:
		return json.Marshal(c)
	case MediaContent:
		return json.Marshal(c)
	case FileContent:
		return json.Marshal(c)
	case VoiceContent:
		return json.Marshal(c)
	case VideoNoteContent:
		return json.Marshal(c)
	case StickerContent:
		return json.Marshal(c)
	case EventContent:
		return json.Marshal(c)
	default:
		return nil, errors.New("unknown content type")
	}
}

func unmarshalContent(msgType MessageType, data []byte) (MessageContent, error) {
	switch msgType {
	case MessageTypeText:
		var c TextContent
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return c, nil
	case MessageTypeImage, MessageTypeVideo:
		var c MediaContent
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return c, nil
	case MessageTypeFile, MessageTypeAudio:
		var c FileContent
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return c, nil
	case MessageTypeVoice:
		var c VoiceContent
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return c, nil
	case MessageTypeVideoNote:
		var c VideoNoteContent
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return c, nil
	case MessageTypeSticker, MessageTypeVideoSticker:
		var c StickerContent
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return c, nil
	case MessageTypeEvent:
		var c EventContent
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return c, nil
	default:
		return nil, errors.New("unknown message type")
	}
}

type pgMessageRepository struct {
	pool *pgxpool.Pool
}

func NewMessageRepository(pool *pgxpool.Pool) MessageRepository {
	return &pgMessageRepository{pool: pool}
}

func (r *pgMessageRepository) CreateMessage(ctx context.Context, input *CreateMessageInput) (*Message, error) {
	contentJSON, err := marshalContent(input.Content)
	if err != nil {
		return nil, fmt.Errorf("marshal content: %w", err)
	}

	const query = `
        INSERT INTO messages (room_id, sender_id, type, content, reply_to_id)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, created_at, updated_at
    `

	var (
		id        uuid.UUID
		createdAt time.Time
		updatedAt time.Time
	)

	err = r.pool.QueryRow(ctx, query,
		input.RoomID,
		input.SenderID,
		input.Type,
		contentJSON,
		input.ReplyToID, // pgx сам обработает nil *uuid.UUID как SQL NULL
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	return &Message{
		ID:        id,
		RoomID:    input.RoomID,
		SenderID:  input.SenderID,
		Type:      input.Type,
		Content:   input.Content,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func (r *pgMessageRepository) GetMessagesByRoomID(ctx context.Context, roomID uuid.UUID, cursor *uuid.UUID, limit int) ([]*Message, *uuid.UUID, error) {
	var rows pgx.Rows
	var err error

	baseQuery := `
		SELECT
			m.id, m.room_id, m.sender_id, m.type, m.content,
			m.reply_to_id, m.deleted_at, m.edited_at, m.created_at, m.updated_at,
			rm.id, rm.sender_id, rm.type, rm.content, rm.deleted_at
		FROM messages m
		LEFT JOIN messages rm ON rm.id = m.reply_to_id
		WHERE m.room_id = $1 AND m.deleted_at IS NULL
	`

	if cursor != nil {
		rows, err = r.pool.Query(ctx, baseQuery+`
			AND (m.created_at, m.id) < (SELECT created_at, id FROM messages WHERE id = $2)
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT $3
		`, roomID, cursor, limit+1)
	} else {
		rows, err = r.pool.Query(ctx, baseQuery+`
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT $2
		`, roomID, limit+1)
	}
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var (
			id, roomID, senderID uuid.UUID
			msgType              string
			contentJSON          []byte
			replyToID            *uuid.UUID
			deletedAt, editedAt  *time.Time
			createdAt, updatedAt time.Time

			rmID, rmSenderID *uuid.UUID
			rmType           *string
			rmContentJSON    []byte
			rmDeletedAt      *time.Time
		)

		err := rows.Scan(
			&id, &roomID, &senderID, &msgType, &contentJSON,
			&replyToID, &deletedAt, &editedAt, &createdAt, &updatedAt,
			&rmID, &rmSenderID, &rmType, &rmContentJSON, &rmDeletedAt,
		)
		if err != nil {
			return nil, nil, err
		}

		content, err := unmarshalContent(MessageType(msgType), contentJSON)
		if err != nil {
			return nil, nil, err
		}

		msg := &Message{
			ID:        id,
			RoomID:    roomID,
			SenderID:  senderID,
			Type:      MessageType(msgType),
			Content:   content,
			DeletedAt: deletedAt,
			EditedAt:  editedAt,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}

		if rmID != nil {
			preview := &MessagePreview{
				MessageID: *rmID,
				SenderID:  *rmSenderID,
			}
			if rmType != nil {
				preview.Type = MessageType(*rmType)
			}

			if rmDeletedAt != nil {
				msg.ReplyTo = preview
			} else if rmContentJSON != nil {
				rmContent, err := unmarshalContent(MessageType(*rmType), rmContentJSON)
				if err == nil {
					switch c := rmContent.(type) {
					case TextContent:
						preview.TextPreview = &c.Text
					case FileContent:
						preview.FileName = &c.FileName
						preview.MimeType = &c.MimeType
					case MediaContent:
						preview.MimeType = &c.MimeType
					case VoiceContent:
						preview.MimeType = &c.MimeType
					case VideoNoteContent:
						preview.MimeType = &c.MimeType
					case StickerContent:
						preview.MimeType = &c.MimeType
					}
				}
				msg.ReplyTo = preview
			}
		}

		messages = append(messages, msg)
	}

	var nextCursor *uuid.UUID
	if len(messages) > limit {
		messages = messages[:limit]
		lastMsg := messages[limit-1]
		nextCursor = &lastMsg.ID
	}

	return messages, nextCursor, nil
}

func (r *pgMessageRepository) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*Message, error) {
	query := `
		SELECT
			m.id, m.room_id, m.sender_id, m.type, m.content,
			m.reply_to_id, m.deleted_at, m.edited_at, m.created_at, m.updated_at,
			rm.id, rm.sender_id, rm.type, rm.content, rm.deleted_at
		FROM messages m
		LEFT JOIN messages rm ON rm.id = m.reply_to_id
		WHERE m.id = $1
	`

	var (
		id, roomID, senderID uuid.UUID
		msgType              string
		contentJSON          []byte
		replyToID            *uuid.UUID
		deletedAt, editedAt  *time.Time
		createdAt, updatedAt time.Time

		rmID, rmSenderID *uuid.UUID
		rmType           *string
		rmContentJSON    []byte
		rmDeletedAt      *time.Time
	)

	err := r.pool.QueryRow(ctx, query, messageID).Scan(
		&id, &roomID, &senderID, &msgType, &contentJSON,
		&replyToID, &deletedAt, &editedAt, &createdAt, &updatedAt,
		&rmID, &rmSenderID, &rmType, &rmContentJSON, &rmDeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	content, err := unmarshalContent(MessageType(msgType), contentJSON)
	if err != nil {
		return nil, err
	}

	msg := &Message{
		ID:        id,
		RoomID:    roomID,
		SenderID:  senderID,
		Type:      MessageType(msgType),
		Content:   content,
		DeletedAt: deletedAt,
		EditedAt:  editedAt,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	if rmID != nil {
		preview := &MessagePreview{
			MessageID: *rmID,
			SenderID:  *rmSenderID,
		}
		if rmType != nil {
			preview.Type = MessageType(*rmType)
		}

		if rmDeletedAt != nil {
			msg.ReplyTo = preview
		} else if rmContentJSON != nil {
			rmContent, err := unmarshalContent(MessageType(*rmType), rmContentJSON)
			if err == nil {
				switch c := rmContent.(type) {
				case TextContent:
					preview.TextPreview = &c.Text
				case FileContent:
					preview.FileName = &c.FileName
					preview.MimeType = &c.MimeType
				case MediaContent:
					preview.MimeType = &c.MimeType
				case VoiceContent:
					preview.MimeType = &c.MimeType
				case VideoNoteContent:
					preview.MimeType = &c.MimeType
				case StickerContent:
					preview.MimeType = &c.MimeType
				}
			}
			msg.ReplyTo = preview
		}
	}

	return msg, nil
}

func (r *pgMessageRepository) SoftDeleteMessage(ctx context.Context, messageID uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE messages
		SET deleted_at = now(), updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
	`, messageID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrMessageNotFound
	}

	return nil
}
