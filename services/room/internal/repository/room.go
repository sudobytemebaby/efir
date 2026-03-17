package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRoomNotFound        = errors.New("room not found")
	ErrRoomAlreadyExists   = errors.New("room already exists")
	ErrMemberNotFound      = errors.New("member not found")
	ErrMemberAlreadyExists = errors.New("member already exists")
	ErrDirectRoomExists    = errors.New("direct room already exists between these users")
)

type RoomType string
type MemberRole string

const (
	RoomTypeDirect RoomType = "direct"
	RoomTypeGroup  RoomType = "group"
)

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleMember MemberRole = "member"
)

type Room struct {
	ID        uuid.UUID
	Name      string
	Type      RoomType
	CreatedBy uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RoomMember struct {
	RoomID   uuid.UUID
	UserID   uuid.UUID
	Role     MemberRole
	JoinedAt time.Time
}

//go:generate mockery --name RoomRepository
type RoomRepository interface {
	CreateRoom(ctx context.Context, name string, roomType RoomType, createdBy uuid.UUID) (*Room, error)
	GetRoomByID(ctx context.Context, id uuid.UUID) (*Room, error)
	UpdateRoom(ctx context.Context, id uuid.UUID, name string) (*Room, error)
	DeleteRoom(ctx context.Context, id uuid.UUID) error
	AddMember(ctx context.Context, roomID, userID uuid.UUID, role MemberRole) (*RoomMember, error)
	RemoveMember(ctx context.Context, roomID, userID uuid.UUID) error
	GetRoomMembers(ctx context.Context, roomID uuid.UUID) ([]RoomMember, error)
	IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
	GetDirectRoomByUsers(ctx context.Context, userID1, userID2 uuid.UUID) (*Room, error)
}

type pgRoomRepository struct {
	pool *pgxpool.Pool
}

func NewRoomRepository(pool *pgxpool.Pool) RoomRepository {
	return &pgRoomRepository{pool: pool}
}

func (r *pgRoomRepository) CreateRoom(ctx context.Context, name string, roomType RoomType, createdBy uuid.UUID) (*Room, error) {
	const query = `
		INSERT INTO rooms (name, type, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, name, type, created_by, created_at, updated_at
	`

	room := &Room{}
	err := r.pool.QueryRow(ctx, query, name, roomType, createdBy).Scan(
		&room.ID, &room.Name, &room.Type, &room.CreatedBy, &room.CreatedAt, &room.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create room: %w", err)
	}

	return room, nil
}

func (r *pgRoomRepository) GetRoomByID(ctx context.Context, id uuid.UUID) (*Room, error) {
	const query = `
		SELECT id, name, type, created_by, created_at, updated_at
		FROM rooms
		WHERE id = $1
	`

	room := &Room{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&room.ID, &room.Name, &room.Type, &room.CreatedBy, &room.CreatedAt, &room.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoomNotFound
		}
		return nil, fmt.Errorf("get room by id: %w", err)
	}

	return room, nil
}

func (r *pgRoomRepository) UpdateRoom(ctx context.Context, id uuid.UUID, name string) (*Room, error) {
	const query = `
		UPDATE rooms
		SET name = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, name, type, created_by, created_at, updated_at
	`

	room := &Room{}
	err := r.pool.QueryRow(ctx, query, id, name).Scan(
		&room.ID, &room.Name, &room.Type, &room.CreatedBy, &room.CreatedAt, &room.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoomNotFound
		}
		return nil, fmt.Errorf("update room: %w", err)
	}

	return room, nil
}

func (r *pgRoomRepository) DeleteRoom(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM rooms WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete room: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrRoomNotFound
	}

	return nil
}

func (r *pgRoomRepository) AddMember(ctx context.Context, roomID, userID uuid.UUID, role MemberRole) (*RoomMember, error) {
	const query = `
		INSERT INTO room_members (room_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (room_id, user_id) DO NOTHING
		RETURNING room_id, user_id, role, joined_at
	`

	member := &RoomMember{}
	err := r.pool.QueryRow(ctx, query, roomID, userID, role).Scan(
		&member.RoomID, &member.UserID, &member.Role, &member.JoinedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMemberAlreadyExists
		}
		return nil, fmt.Errorf("add member: %w", err)
	}

	return member, nil
}

func (r *pgRoomRepository) RemoveMember(ctx context.Context, roomID, userID uuid.UUID) error {
	const query = `DELETE FROM room_members WHERE room_id = $1 AND user_id = $2`

	result, err := r.pool.Exec(ctx, query, roomID, userID)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrMemberNotFound
	}

	return nil
}

func (r *pgRoomRepository) GetRoomMembers(ctx context.Context, roomID uuid.UUID) ([]RoomMember, error) {
	const query = `
		SELECT room_id, user_id, role, joined_at
		FROM room_members
		WHERE room_id = $1
	`

	rows, err := r.pool.Query(ctx, query, roomID)
	if err != nil {
		return nil, fmt.Errorf("get room members: %w", err)
	}
	defer rows.Close()

	var members []RoomMember
	for rows.Next() {
		var member RoomMember
		if err := rows.Scan(&member.RoomID, &member.UserID, &member.Role, &member.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate members: %w", err)
	}

	return members, nil
}

func (r *pgRoomRepository) IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	const query = `
		SELECT 1 FROM room_members
		WHERE room_id = $1 AND user_id = $2
		LIMIT 1
	`

	var exists int
	err := r.pool.QueryRow(ctx, query, roomID, userID).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check membership: %w", err)
	}

	return true, nil
}

func (r *pgRoomRepository) GetDirectRoomByUsers(ctx context.Context, userID1, userID2 uuid.UUID) (*Room, error) {
	const query = `
		SELECT r.id, r.name, r.type, r.created_by, r.created_at, r.updated_at
		FROM rooms r
		JOIN room_members rm1 ON r.id = rm1.room_id AND rm1.user_id = $1
		JOIN room_members rm2 ON r.id = rm2.room_id AND rm2.user_id = $2
		WHERE r.type = 'direct'
		LIMIT 1
	`

	room := &Room{}
	err := r.pool.QueryRow(ctx, query, userID1, userID2).Scan(
		&room.ID, &room.Name, &room.Type, &room.CreatedBy, &room.CreatedAt, &room.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoomNotFound
		}
		return nil, fmt.Errorf("get direct room by users: %w", err)
	}

	return room, nil
}
