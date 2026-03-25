package room

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler"
	"github.com/sudobytemebaby/efir/services/gateway/internal/middleware"
	roomv1 "github.com/sudobytemebaby/efir/services/shared/gen/room"
)

type Handler struct {
	client roomv1.RoomServiceClient
}

func NewHandler(client roomv1.RoomServiceClient) *Handler {
	return &Handler{client: client}
}

func (h *Handler) Register(r chi.Router) {
	r.Post("/rooms", h.createRoom)
	r.Get("/rooms/{id}", h.getRoom)
	r.Patch("/rooms/{id}", h.updateRoom)
	r.Delete("/rooms/{id}", h.deleteRoom)
	r.Post("/rooms/{id}/members", h.addMember)
	r.Delete("/rooms/{id}/members/{userId}", h.removeMember)
}

func (h *Handler) createRoom(w http.ResponseWriter, r *http.Request) {
	var req roomv1.CreateRoomRequest
	if err := handler.ReadProto(r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.CreatedBy = middleware.MustGetUserID(r.Context())

	resp, err := h.client.CreateRoom(middleware.InjectRequestIDToOutgoingContext(r.Context()), &req)
	if err != nil {
		handler.WriteError(w, r, err, "failed to create room")
		return
	}
	handler.WriteProto(w, http.StatusCreated, resp.Room)
}

func (h *Handler) getRoom(w http.ResponseWriter, r *http.Request) {
	resp, err := h.client.GetRoom(
		middleware.InjectRequestIDToOutgoingContext(r.Context()),
		&roomv1.GetRoomRequest{RoomId: chi.URLParam(r, "id")},
	)
	if err != nil {
		handler.WriteError(w, r, err, "failed to get room")
		return
	}
	handler.WriteProto(w, http.StatusOK, resp.Room)
}

func (h *Handler) updateRoom(w http.ResponseWriter, r *http.Request) {
	var req roomv1.UpdateRoomRequest
	if err := handler.ReadProto(r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.RoomId = chi.URLParam(r, "id")
	req.RequesterId = middleware.MustGetUserID(r.Context())

	resp, err := h.client.UpdateRoom(middleware.InjectRequestIDToOutgoingContext(r.Context()), &req)
	if err != nil {
		handler.WriteError(w, r, err, "failed to update room")
		return
	}
	handler.WriteProto(w, http.StatusOK, resp.Room)
}

func (h *Handler) deleteRoom(w http.ResponseWriter, r *http.Request) {
	_, err := h.client.DeleteRoom(
		middleware.InjectRequestIDToOutgoingContext(r.Context()),
		&roomv1.DeleteRoomRequest{
			RoomId:      chi.URLParam(r, "id"),
			RequesterId: middleware.MustGetUserID(r.Context()),
		},
	)
	if err != nil {
		handler.WriteError(w, r, err, "failed to delete room")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) {
	var req roomv1.AddMemberRequest
	if err := handler.ReadProto(r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.RoomId = chi.URLParam(r, "id")
	req.RequesterId = middleware.MustGetUserID(r.Context())

	if _, err := h.client.AddMember(middleware.InjectRequestIDToOutgoingContext(r.Context()), &req); err != nil {
		handler.WriteError(w, r, err, "failed to add member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	_, err := h.client.RemoveMember(
		middleware.InjectRequestIDToOutgoingContext(r.Context()),
		&roomv1.RemoveMemberRequest{
			RoomId:      chi.URLParam(r, "id"),
			UserId:      chi.URLParam(r, "userId"),
			RequesterId: middleware.MustGetUserID(r.Context()),
		},
	)
	if err != nil {
		handler.WriteError(w, r, err, "failed to remove member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
