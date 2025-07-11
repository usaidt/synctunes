package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"

	"synctunes/internal/handlers"
	"synctunes/internal/music"
	"synctunes/internal/room"
	"synctunes/internal/websocket"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	musicDir := os.Getenv("MUSIC_DIR")
	if musicDir == "" {
		musicDir = "./music"
	}

	// Ensure music directory exists
	if err := os.MkdirAll(musicDir, 0755); err != nil {
		log.Fatal("Failed to create music directory:", err)
	}

	// Initialize services
	musicService := music.NewService(musicDir)
	roomManager := room.NewManager()
	wsHub := websocket.NewHub(roomManager)

	// Start WebSocket hub
	go wsHub.Run()

	// Initialize handlers
	h := handlers.New(musicService, roomManager, wsHub)

	// Setup routes
	r := mux.NewRouter()
	
	// API routes
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/music/catalog", h.GetMusicCatalog).Methods("GET")
	api.HandleFunc("/music/stream/{id}", h.StreamMusic).Methods("GET")
	api.HandleFunc("/rooms", h.CreateRoom).Methods("POST")
	api.HandleFunc("/rooms/{id}", h.GetRoom).Methods("GET")
	api.HandleFunc("/rooms/{id}/join", h.JoinRoom).Methods("POST")
	api.HandleFunc("/rooms/{id}/play", h.PlayTrack).Methods("POST")
	api.HandleFunc("/rooms/{id}/pause", h.PauseRoom).Methods("POST")
	api.HandleFunc("/rooms/{id}/resume", h.ResumeRoom).Methods("POST")
	api.HandleFunc("/rooms/{id}/seek", h.SeekTrack).Methods("POST")

	// WebSocket endpoint
	r.HandleFunc("/ws/{roomId}", h.HandleWebSocket)

	// Static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/"))))

	// Main page
	r.HandleFunc("/", h.HomePage).Methods("GET")
	r.HandleFunc("/room/{id}", h.RoomPage).Methods("GET")
	r.HandleFunc("/listen/{id}", h.ListenerPage).Methods("GET")

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(r)

	log.Printf("Server starting on port %s", port)
	log.Printf("Music directory: %s", musicDir)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
