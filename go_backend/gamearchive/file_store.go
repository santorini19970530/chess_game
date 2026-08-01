// CM3070 FP code
// file_store.go - json file adapter for archived game history

package gamearchive

import (
	"encoding/json"
	"os"
	"path/filepath"

	sessionpkg "go_backend/game/session"
)

// JSONFileStore - persists archived games as json under a file path
type JSONFileStore struct {
	Path string
}

// ResolvePath - picks game_history.json next to the binary, else user cache, else cwd/data
func ResolvePath() string {
	if execPath, err := os.Executable(); err == nil {
		if execPath != "" && execPath != "." {
			return filepath.Join(filepath.Dir(execPath), "data", "game_history.json")
		}
	}
	if cacheDir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(cacheDir, "chess_game", "data", "game_history.json")
	}
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Join(cwd, "data", "game_history.json")
	}
	return filepath.Join("data", "game_history.json")
}

// NewJSONFileStore - creates a json archive store at path
func NewJSONFileStore(path string) *JSONFileStore {
	return &JSONFileStore{Path: path}
}

// LoadArchivedGames - reads archived games from disk (missing file → empty list)
func (s *JSONFileStore) LoadArchivedGames() ([]sessionpkg.ArchivedGame, error) {
	bytes, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return []sessionpkg.ArchivedGame{}, nil
		}
		return nil, err
	}
	if len(bytes) == 0 {
		return []sessionpkg.ArchivedGame{}, nil
	}
	var records []sessionpkg.ArchivedGame
	if err := json.Unmarshal(bytes, &records); err != nil {
		return nil, err
	}
	return records, nil
}

// SaveArchivedGames - writes archived games to disk as indented json
func (s *JSONFileStore) SaveArchivedGames(records []sessionpkg.ArchivedGame) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.Path, bytes, 0o644)
}
