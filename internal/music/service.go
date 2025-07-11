package music

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
)

type Track struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Duration int    `json:"duration"` // in seconds
	IsVideo  bool   `json:"is_video"` // true if this is a video file
	Path     string `json:"-"`        // don't expose file path
}

type Service struct {
	musicDir string
	catalog  []Track
}

func NewService(musicDir string) *Service {
	s := &Service{
		musicDir: musicDir,
		catalog:  make([]Track, 0),
	}
	s.scanMusicDirectory()
	return s
}

func (s *Service) GetCatalog() []Track {
	return s.catalog
}

func (s *Service) GetTrack(id string) (*Track, error) {
	for _, track := range s.catalog {
		if track.ID == id {
			return &track, nil
		}
	}
	return nil, fmt.Errorf("track not found")
}

func (s *Service) scanMusicDirectory() {
	log.Printf("Scanning music directory: %s", s.musicDir)
	
	err := filepath.WalkDir(s.musicDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Check for audio and video file extensions
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".mp3" || ext == ".wav" || ext == ".flac" || ext == ".ogg" || ext == ".m4a" ||
		   ext == ".mp4" || ext == ".mkv" || ext == ".avi" || ext == ".mov" || ext == ".webm" || ext == ".wmv" {
			track := s.createTrackFromPath(path)
			s.catalog = append(s.catalog, track)
		}

		return nil
	})

	if err != nil {
		log.Printf("Error scanning music directory: %v", err)
	}

	log.Printf("Found %d tracks", len(s.catalog))
}

func (s *Service) createTrackFromPath(path string) Track {
	// Extract filename without extension
	filename := filepath.Base(path)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	
	// Check if it's a video file
	ext := strings.ToLower(filepath.Ext(path))
	isVideo := ext == ".mp4" || ext == ".mkv" || ext == ".avi" || ext == ".mov" || ext == ".webm" || ext == ".wmv"
	
	// Simple parsing - try to extract artist and title from filename
	// Format: "Artist - Title" or just "Title"
	parts := strings.Split(name, " - ")
	
	var artist, title string
	if len(parts) >= 2 {
		artist = strings.TrimSpace(parts[0])
		title = strings.TrimSpace(strings.Join(parts[1:], " - "))
	} else {
		artist = "Unknown Artist"
		title = name
	}

	// Use relative path from music directory as ID
	relPath, _ := filepath.Rel(s.musicDir, path)
	id := strings.ReplaceAll(relPath, "\\", "/") // normalize path separators

	return Track{
		ID:       id,
		Title:    title,
		Artist:   artist,
		Album:    "Unknown Album",
		Duration: 0, // TODO: extract actual duration from file metadata
		IsVideo:  isVideo,
		Path:     path,
	}
}

func (s *Service) RescanCatalog() {
	s.catalog = make([]Track, 0)
	s.scanMusicDirectory()
}

func (s *Service) GetCatalogJSON() ([]byte, error) {
	return json.Marshal(s.catalog)
}
