package session

import (
	"fmt"
	"sync"

	pieces "go_backend/game/piece"
)

type RuntimeState struct {
	Pieces             []pieces.ChessPiece
	MoveHistory        []string
	MoveHistoryDetailed []MoveHistoryEntry
	CurrentTurnOverride *pieces.PieceColor
	CurrentTurnPinned   bool
	HalfmoveClock       int
	PositionCounts      map[string]int
	LastAppliedMove     *LastMove
	WhiteKingMoved      bool
	BlackKingMoved      bool
	WhiteRookAMoved     bool
	WhiteRookHMoved     bool
	BlackRookAMoved     bool
	BlackRookHMoved     bool
}

type RuntimeGame struct {
	Session GameSession
	State   RuntimeState
}

type SessionStore struct {
	mu    sync.RWMutex
	games map[string]*RuntimeGame
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		games: make(map[string]*RuntimeGame),
	}
}

func (s *SessionStore) Create(session GameSession) *RuntimeGame {
	s.mu.Lock()
	defer s.mu.Unlock()
	game := &RuntimeGame{
		Session: session,
		State:   newInitialRuntimeState(),
	}
	s.games[session.ID] = game
	return game
}

func (s *SessionStore) Get(gameID string) (*RuntimeGame, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	game, ok := s.games[gameID]
	return game, ok
}

func (s *SessionStore) Update(gameID string, updater func(*RuntimeGame) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	game, ok := s.games[gameID]
	if !ok {
		return fmt.Errorf("game session not found: %s", gameID)
	}
	return updater(game)
}

func (s *SessionStore) Delete(gameID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.games[gameID]; !ok {
		return false
	}
	delete(s.games, gameID)
	return true
}

func newInitialRuntimeState() RuntimeState {
	return RuntimeState{
		Pieces:        append([]pieces.ChessPiece(nil), initialPiecesSnapshot...),
		PositionCounts: make(map[string]int),
	}
}

func (g *RuntimeGame) bindToGlobals() {
	pieces.ChessPieces = append([]pieces.ChessPiece(nil), g.State.Pieces...)
	moveHistory = append([]string(nil), g.State.MoveHistory...)
	moveHistoryDetailed = append([]MoveHistoryEntry(nil), g.State.MoveHistoryDetailed...)
	if g.State.CurrentTurnOverride != nil {
		c := *g.State.CurrentTurnOverride
		currentTurnOverride = &c
	} else {
		currentTurnOverride = nil
	}
	currentTurnPinned = g.State.CurrentTurnPinned
	halfmoveClock = g.State.HalfmoveClock
	positionCounts = copyStringIntMap(g.State.PositionCounts)
	if g.State.LastAppliedMove != nil {
		mv := *g.State.LastAppliedMove
		lastAppliedMove = &mv
	} else {
		lastAppliedMove = nil
	}
	whiteKingMoved = g.State.WhiteKingMoved
	blackKingMoved = g.State.BlackKingMoved
	whiteRookAMoved = g.State.WhiteRookAMoved
	whiteRookHMoved = g.State.WhiteRookHMoved
	blackRookAMoved = g.State.BlackRookAMoved
	blackRookHMoved = g.State.BlackRookHMoved
}

func (g *RuntimeGame) syncFromGlobals() {
	g.State.Pieces = append([]pieces.ChessPiece(nil), pieces.ChessPieces...)
	g.State.MoveHistory = append([]string(nil), moveHistory...)
	g.State.MoveHistoryDetailed = append([]MoveHistoryEntry(nil), moveHistoryDetailed...)
	if currentTurnOverride != nil {
		c := *currentTurnOverride
		g.State.CurrentTurnOverride = &c
	} else {
		g.State.CurrentTurnOverride = nil
	}
	g.State.CurrentTurnPinned = currentTurnPinned
	g.State.HalfmoveClock = halfmoveClock
	g.State.PositionCounts = copyStringIntMap(positionCounts)
	if lastAppliedMove != nil {
		mv := *lastAppliedMove
		g.State.LastAppliedMove = &mv
	} else {
		g.State.LastAppliedMove = nil
	}
	g.State.WhiteKingMoved = whiteKingMoved
	g.State.BlackKingMoved = blackKingMoved
	g.State.WhiteRookAMoved = whiteRookAMoved
	g.State.WhiteRookHMoved = whiteRookHMoved
	g.State.BlackRookAMoved = blackRookAMoved
	g.State.BlackRookHMoved = blackRookHMoved
}

func copyStringIntMap(in map[string]int) map[string]int {
	if in == nil {
		return make(map[string]int)
	}
	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
