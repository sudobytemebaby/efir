package room

type createRoomRequest struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	ParticipantID string `json:"participant_id,omitempty"`
}

type roomResponse struct {
	RoomID    string `json:"room_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type updateRoomRequest struct {
	Name *string `json:"name,omitempty"`
}

type memberRequest struct {
	UserID string `json:"user_id"`
}
