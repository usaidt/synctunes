package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"synctunes/internal/music"
	"synctunes/internal/room"
	"synctunes/internal/websocket"
)

type Handler struct {
	musicService *music.Service
	roomManager  *room.Manager
	wsHub        *websocket.Hub
	templates    *template.Template
}

type CreateRoomRequest struct {
	Name string `json:"name"`
}

type JoinRoomRequest struct {
	UserName string `json:"user_name"`
}

type PlayTrackRequest struct {
	TrackID string `json:"track_id"`
	UserID  string `json:"user_id"`
}

type SeekRequest struct {
	Position int    `json:"position"`
	UserID   string `json:"user_id"`
}

type PlaybackControlRequest struct {
	UserID string `json:"user_id"`
}

func New(musicService *music.Service, roomManager *room.Manager, wsHub *websocket.Hub) *Handler {
	// Define custom template functions
	funcMap := template.FuncMap{
		"json": func(v interface{}) template.JS {
			bytes, err := json.Marshal(v)
			if err != nil {
				return template.JS("{}")
			}
			return template.JS(bytes)
		},
	}
	
	// Load templates with custom functions
	templates := template.New("").Funcs(funcMap)
	templates = template.Must(templates.ParseGlob("web/templates/*.html"))
	
	return &Handler{
		musicService: musicService,
		roomManager:  roomManager,
		wsHub:        wsHub,
		templates:    templates,
	}
}

func (h *Handler) HomePage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title string
	}{
		Title: "SyncTunes - Collaborative Music Streaming",
	}
	
	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func (h *Handler) RoomPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]
	
	room, exists := h.roomManager.GetRoom(roomID)
	if !exists {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}
	
	// Check if this is the host by looking at the host_id query parameter
	hostID := r.URL.Query().Get("host_id")
	isHost := hostID != "" && room.IsHost(hostID)
	
	data := struct {
		Title  string
		Room   interface{}
		RoomID string
		IsHost bool
		HostID string
	}{
		Title:  fmt.Sprintf("Room: %s", room.Name),
		Room:   room.GetState(),
		RoomID: roomID,
		IsHost: isHost,
		HostID: hostID,
	}
	
	templateName := "room.html"
	if !isHost {
		templateName = "listener.html"
	}
	
	if err := h.templates.ExecuteTemplate(w, templateName, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func (h *Handler) ListenerPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]
	
	room, exists := h.roomManager.GetRoom(roomID)
	if !exists {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}
	
	data := struct {
		Title  string
		Room   interface{}
		RoomID string
		IsHost bool
	}{
		Title:  fmt.Sprintf("Listening to: %s", room.Name),
		Room:   room.GetState(),
		RoomID: roomID,
		IsHost: false,
	}
	
	if err := h.templates.ExecuteTemplate(w, "listener.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func (h *Handler) GetMusicCatalog(w http.ResponseWriter, r *http.Request) {
	catalog := h.musicService.GetCatalog()
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(catalog); err != nil {
		http.Error(w, "Error encoding catalog", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) StreamMusic(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	trackID := vars["id"]
	
	track, err := h.musicService.GetTrack(trackID)
	if err != nil {
		http.Error(w, "Track not found", http.StatusNotFound)
		return
	}
	
	file, err := os.Open(track.Path)
	if err != nil {
		http.Error(w, "Error opening file", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	
	// Get file info for size
	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Error getting file info", http.StatusInternalServerError)
		return
	}
	
	// Set headers for audio/video streaming
	ext := strings.ToLower(filepath.Ext(track.Path))
	var contentType string
	switch ext {
	case ".mp3":
		contentType = "audio/mpeg"
	case ".wav":
		contentType = "audio/wav"
	case ".flac":
		contentType = "audio/flac"
	case ".ogg":
		contentType = "audio/ogg"
	case ".m4a":
		contentType = "audio/mp4"
	case ".mp4":
		contentType = "video/mp4"
	case ".mkv":
		contentType = "video/x-matroska"
	case ".avi":
		contentType = "video/x-msvideo"
	case ".mov":
		contentType = "video/quicktime"
	case ".webm":
		contentType = "video/webm"
	case ".wmv":
		contentType = "video/x-ms-wmv"
	default:
		contentType = "audio/mpeg"
	}
	
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	w.Header().Set("Accept-Ranges", "bytes")
	
	// Handle range requests for seeking
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		h.handleRangeRequest(w, r, file, stat.Size())
		return
	}
	
	// Stream the entire file
	io.Copy(w, file)
}

func (h *Handler) handleRangeRequest(w http.ResponseWriter, r *http.Request, file *os.File, fileSize int64) {
	// Simple range request handling
	// In a production app, you'd want more robust range parsing
	w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", fileSize-1, fileSize))
	w.WriteHeader(http.StatusPartialContent)
	io.Copy(w, file)
}

func (h *Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	roomID := uuid.New().String()
	hostID := uuid.New().String() // In a real app, this would come from auth
	
	room := h.roomManager.CreateRoom(roomID, req.Name, hostID)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"room_id": roomID,
		"host_id": hostID,
		"room_name": room.Name,
	})
}

func (h *Handler) GetRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]
	
	room, exists := h.roomManager.GetRoom(roomID)
	if !exists {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	roomJSON, _ := room.ToJSON()
	w.Write(roomJSON)
}

func (h *Handler) JoinRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]
	
	var req JoinRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	userID := uuid.New().String() // In a real app, this would come from auth
	
	if err := h.roomManager.JoinRoom(roomID, userID, req.UserName); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	// Broadcast room update
	if room, exists := h.roomManager.GetRoom(roomID); exists {
		roomJSON, _ := room.ToJSON()
		h.wsHub.BroadcastToRoom(roomID, roomJSON)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"user_id": userID,
	})
}

func (h *Handler) PlayTrack(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]
	
	var req PlayTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	room, exists := h.roomManager.GetRoom(roomID)
	if !exists {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}
	
	// Check if user has permission to control playback
	if !room.CanControlPlayback(req.UserID) {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}
	
	track, err := h.musicService.GetTrack(req.TrackID)
	if err != nil {
		http.Error(w, "Track not found", http.StatusNotFound)
		return
	}
	
	room.PlayTrack(track)
	
	// Broadcast room update
	roomJSON, _ := room.ToJSON()
	h.wsHub.BroadcastToRoom(roomID, roomJSON)
	
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) PauseRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]
	
	var req PlaybackControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	room, exists := h.roomManager.GetRoom(roomID)
	if !exists {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}
	
	// Check if user has permission to control playback
	if !room.CanControlPlayback(req.UserID) {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}
	
	room.Pause()
	
	// Broadcast room update
	roomJSON, _ := room.ToJSON()
	h.wsHub.BroadcastToRoom(roomID, roomJSON)
	
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ResumeRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]
	
	var req PlaybackControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	room, exists := h.roomManager.GetRoom(roomID)
	if !exists {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}
	
	// Check if user has permission to control playback
	if !room.CanControlPlayback(req.UserID) {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}
	
	room.Resume()
	
	// Broadcast room update
	roomJSON, _ := room.ToJSON()
	h.wsHub.BroadcastToRoom(roomID, roomJSON)
	
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) SeekTrack(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]
	
	var req SeekRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	room, exists := h.roomManager.GetRoom(roomID)
	if !exists {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}
	
	// Check if user has permission to control playback
	if !room.CanControlPlayback(req.UserID) {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}
	
	room.Seek(req.Position)
	
	// Broadcast room update
	roomJSON, _ := room.ToJSON()
	h.wsHub.BroadcastToRoom(roomID, roomJSON)
	
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["roomId"]
	
	// In a real app, you'd get userID from authentication
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = uuid.New().String()
	}
	
	h.wsHub.HandleWebSocket(w, r, roomID, userID)
}
