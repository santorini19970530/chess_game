package session

import "sync"

// ArchiveStore - persistence port for ended-game snapshots (implemented outside session)
type ArchiveStore interface {
	LoadArchivedGames() ([]ArchivedGame, error)
	SaveArchivedGames(records []ArchivedGame) error
}

type memoryArchiveStore struct {
	mu      sync.Mutex
	records []ArchivedGame
}

// LoadArchivedGames - returns in-memory archived games
func (m *memoryArchiveStore) LoadArchivedGames() ([]ArchivedGame, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]ArchivedGame, len(m.records))
	copy(out, m.records)
	return out, nil
}

// SaveArchivedGames - replaces in-memory archived games
func (m *memoryArchiveStore) SaveArchivedGames(records []ArchivedGame) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = make([]ArchivedGame, len(records))
	copy(m.records, records)
	return nil
}

var (
	archiveStoreMu sync.RWMutex
	archiveStore   ArchiveStore = &memoryArchiveStore{}
)

// SetArchiveStore - wires the archive persistence adapter (call from main / tests)
func SetArchiveStore(store ArchiveStore) {
	archiveStoreMu.Lock()
	defer archiveStoreMu.Unlock()
	if store == nil {
		archiveStore = &memoryArchiveStore{}
		return
	}
	archiveStore = store
}

// loadArchivedGames - loads via the configured archive store
func loadArchivedGames() ([]ArchivedGame, error) {
	archiveStoreMu.RLock()
	store := archiveStore
	archiveStoreMu.RUnlock()
	return store.LoadArchivedGames()
}

// saveArchivedGames - saves via the configured archive store
func saveArchivedGames(records []ArchivedGame) error {
	archiveStoreMu.RLock()
	store := archiveStore
	archiveStoreMu.RUnlock()
	return store.SaveArchivedGames(records)
}
