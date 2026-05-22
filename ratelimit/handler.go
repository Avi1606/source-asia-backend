package ratelimit

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	store *RateLimitStore
}

func NewHandler(store *RateLimitStore) *Handler {
	return &Handler{store: store}
}

type requestBody struct {
	UserID  string          `json:"user_id"`
	Payload json.RawMessage `json:"payload"`
}

type requestAcceptedResponse struct {
	Status    string `json:"status"`
	UserID    string `json:"user_id"`
	Timestamp string `json:"timestamp"`
}

type requestRejectedResponse struct {
	Error  string `json:"error"`
	UserID string `json:"user_id"`
	Limit  int    `json:"limit"`
	Window string `json:"window"`
}

type userStatsResponse struct {
	AcceptedCurrentWindow int `json:"accepted_current_window"`
	RejectedTotal         int `json:"rejected_total"`
}

type statsResponse struct {
	Users  map[string]userStatsResponse `json:"users"`
	Global globalStatsResponse          `json:"global"`
}

type globalStatsResponse struct {
	TotalAccepted int `json:"total_accepted"`
	TotalRejected int `json:"total_rejected"`
}

func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	var body requestBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	userID := strings.TrimSpace(body.UserID)
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id is required and must be non-empty"})
		return
	}
	if len(body.Payload) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "payload is required"})
		return
	}

	if !h.store.Allow(userID) {
		writeJSON(w, http.StatusTooManyRequests, requestRejectedResponse{
			Error:  "rate limit exceeded",
			UserID: userID,
			Limit:  RequestLimit,
			Window: "1m",
		})
		return
	}

	writeJSON(w, http.StatusCreated, requestAcceptedResponse{
		Status:    "accepted",
		UserID:    userID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) HandleStats(w http.ResponseWriter, r *http.Request) {
	snapshot := h.store.Stats()
	users := make(map[string]userStatsResponse, len(snapshot))
	var totalAccepted int
	var totalRejected int

	for userID, state := range snapshot {
		users[userID] = userStatsResponse{
			AcceptedCurrentWindow: state.AcceptedCount,
			RejectedTotal:         state.RejectedTotal,
		}
		totalAccepted += state.AcceptedCount
		totalRejected += state.RejectedTotal
	}

	writeJSON(w, http.StatusOK, statsResponse{
		Users: users,
		Global: globalStatsResponse{
			TotalAccepted: totalAccepted,
			TotalRejected: totalRejected,
		},
	})
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(dst); err != nil {
		return errors.New("invalid JSON body")
	}

	var extra struct{}
	if err := decoder.Decode(&extra); err == nil {
		return errors.New("request body must contain a single JSON object")
	} else if !errors.Is(err, io.EOF) {
		return errors.New("invalid JSON body")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
