package ratelimit

import (
	"sync"
	"time"
)

const (
	RequestLimit = 5
	WindowSize   = time.Minute
)

type UserState struct {
	WindowStart   time.Time
	AcceptedCount int
	RejectedTotal int
}

type RateLimitStore struct {
	mu    sync.Mutex
	users map[string]*UserState
}

func NewRateLimitStore() *RateLimitStore {
	return &RateLimitStore{
		users: make(map[string]*UserState),
	}
}

func (s *RateLimitStore) Allow(userID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	state, ok := s.users[userID]
	if !ok {
		s.users[userID] = &UserState{
			WindowStart:   now,
			AcceptedCount: 1,
		}
		return true
	}

	if now.After(state.WindowStart.Add(WindowSize)) {
		state.WindowStart = now
		state.AcceptedCount = 0
	}

	if state.AcceptedCount >= RequestLimit {
		state.RejectedTotal++
		return false
	}

	state.AcceptedCount++
	return true
}

func (s *RateLimitStore) Stats() map[string]UserState {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := make(map[string]UserState, len(s.users))
	for userID, state := range s.users {
		snapshot[userID] = *state
	}
	return snapshot
}
