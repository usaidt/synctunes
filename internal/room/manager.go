package room

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"synctunes/internal/music"
)

type PlaybackState string

const (
	StatePlaying PlaybackState = "playing"
	StatePaused  PlaybackState = "paused"
	StateStopped PlaybackState = "stopped"
)

type UserRole string

const (
	RoleHost     UserRole = "host"
	RoleListener UserRole = "listener"
)

type Room struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	CurrentTrack  *music.Track        `json:"current_track"`
	State         PlaybackState       `json:"state"`
	Position      int                 `json:"position"` // current position in seconds
	LastUpdate    time.Time           `json:"last_update"`
	Listeners     map[string]*User    `json:"listeners"`
	Host          string              `json:"host"`
	CreatedAt     time.Time           `json:"created_at"`
	mu            sync.RWMutex        `json:"-"`
}
  
type User struct {
	ID   string   `json:"id"`
	Name string   `json:"name"`
	Role UserRole `json:"role"`
}

type Manager struct {
	rooms map[string]*Room
	mu    sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		rooms: make(map[string]*Room),
	}
}

func (m *Manager) CreateRoom(id, name, hostID string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()

	room := &Room{
		ID:         id,
		Name:       name,
		State:      StateStopped,
		Position:   0,
		LastUpdate: time.Now(),
		Listeners:  make(map[string]*User),
		Host:       hostID,
		CreatedAt:  time.Now(),
	}

	// Add the host as a user
	room.Listeners[hostID] = &User{
		ID:   hostID,
		Name: "Host",
		Role: RoleHost,
	}

	m.rooms[id] = room
	return room
}

func (m *Manager) GetRoom(id string) (*Room, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	room, exists := m.rooms[id]
	return room, exists
}

func (m *Manager) DeleteRoom(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.rooms, id)
}

func (m *Manager) JoinRoom(roomID, userID, userName string) error {
	m.mu.RLock()
	room, exists := m.rooms[roomID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("room not found")
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	room.Listeners[userID] = &User{
		ID:   userID,
		Name: userName,
		Role: RoleListener,
	}

	return nil
}

func (m *Manager) LeaveRoom(roomID, userID string) error {
	m.mu.RLock()
	room, exists := m.rooms[roomID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("room not found")
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	delete(room.Listeners, userID)
	return nil
}

func (r *Room) CanControlPlayback(userID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	user, exists := r.Listeners[userID]
	if !exists {
		return false
	}
	
	return user.Role == RoleHost
}

func (r *Room) IsHost(userID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return r.Host == userID
}

func (r *Room) GetUserRole(userID string) UserRole {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if user, exists := r.Listeners[userID]; exists {
		return user.Role
	}
	return RoleListener
}

func (r *Room) PlayTrack(track *music.Track) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.CurrentTrack = track
	r.State = StatePlaying
	r.Position = 0
	r.LastUpdate = time.Now()
}

func (r *Room) Pause() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.State == StatePlaying {
		// Update position based on time elapsed
		elapsed := time.Since(r.LastUpdate).Seconds()
		r.Position += int(elapsed)
		r.State = StatePaused
		r.LastUpdate = time.Now()
	}
}

func (r *Room) Resume() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.State == StatePaused {
		r.State = StatePlaying
		r.LastUpdate = time.Now()
	}
}

func (r *Room) Seek(position int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Position = position
	r.LastUpdate = time.Now()
}

func (r *Room) GetCurrentPosition() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.State == StatePlaying {
		elapsed := time.Since(r.LastUpdate).Seconds()
		return r.Position + int(elapsed)
	}
	return r.Position
}

func (r *Room) GetState() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	currentPosition := r.Position
	if r.State == StatePlaying {
		elapsed := time.Since(r.LastUpdate).Seconds()
		currentPosition += int(elapsed)
	}

	listeners := make([]User, 0, len(r.Listeners))
	for _, user := range r.Listeners {
		listeners = append(listeners, *user)
	}

	return map[string]interface{}{
		"id":             r.ID,
		"name":           r.Name,
		"current_track":  r.CurrentTrack,
		"state":          r.State,
		"position":       currentPosition,
		"listeners":      listeners,
		"host":           r.Host,
		"created_at":     r.CreatedAt,
	}
}

func (r *Room) ToJSON() ([]byte, error) {
	state := r.GetState()
	return json.Marshal(state)
}
