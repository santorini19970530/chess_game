// CM3070 FP code
// store.go - in-memory game session store

package session

import (
	"fmt"
	"sync"

	pieces "go_backend/game/piece"
)

// runtime state for the game
type RuntimeState struct {
	Pieces              []pieces.ChessPiece
	MoveHistory         []string
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
	// boardFEN is source of truth for non-chess variants (e.g. xiangqi). empty for chess.
	BoardFEN              string
	XiangqiIdlePly        int
	XiangqiPositionCounts map[string]int
	XiangqiPositionKeys   []string
	XiangqiPlyFacts       []xiangqiPlyFact
	ShogiPositionCounts   map[string]int
	ShogiPositionKeys     []string
	ShogiPlyFacts         []shogiPlyFact
}

// runtime game for the game
type RuntimeGame struct {
	Session GameSession
	State   RuntimeState
}

// session store for the game
type SessionStore struct {
	mu    sync.RWMutex
	games map[string]*RuntimeGame
}

// NewSessionStore - creates session store
func NewSessionStore() *SessionStore {
	return &SessionStore{
		games: make(map[string]*RuntimeGame),
	}
}

// Create - creates the operation
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

// Get - returns the value
func (s *SessionStore) Get(gameID string) (*RuntimeGame, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	game, ok := s.games[gameID]
	return game, ok
}

// Update - updates the operation
func (s *SessionStore) Update(gameID string, updater func(*RuntimeGame) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	game, ok := s.games[gameID]
	if !ok {
		return fmt.Errorf("game session not found: %s", gameID)
	}
	return updater(game)
}

// Delete - deletes the operation
func (s *SessionStore) Delete(gameID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.games[gameID]; !ok {
		return false
	}
	delete(s.games, gameID)
	return true
}

// newInitialRuntimeState - creates initial runtime state
func newInitialRuntimeState() RuntimeState {
	return RuntimeState{
		Pieces:                append([]pieces.ChessPiece(nil), initialPiecesSnapshot...),
		PositionCounts:        make(map[string]int),
		XiangqiPositionCounts: make(map[string]int),
		ShogiPositionCounts:   make(map[string]int),
	}
}

// bindToGlobals - binds to globals
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
	boardFEN = g.State.BoardFEN
	xiangqiIdlePly = g.State.XiangqiIdlePly
	xiangqiPositionCounts = copyStringIntMap(g.State.XiangqiPositionCounts)
	xiangqiPositionKeys = append([]string(nil), g.State.XiangqiPositionKeys...)
	xiangqiPlyFacts = copyXiangqiPlyFacts(g.State.XiangqiPlyFacts)
	shogiPositionCounts = copyStringIntMap(g.State.ShogiPositionCounts)
	shogiPositionKeys = append([]string(nil), g.State.ShogiPositionKeys...)
	shogiPlyFacts = copyShogiPlyFacts(g.State.ShogiPlyFacts)
}

// syncFromGlobals - syncs from globals
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
	g.State.BoardFEN = boardFEN
	g.State.XiangqiIdlePly = xiangqiIdlePly
	g.State.XiangqiPositionCounts = copyStringIntMap(xiangqiPositionCounts)
	g.State.XiangqiPositionKeys = append([]string(nil), xiangqiPositionKeys...)
	g.State.XiangqiPlyFacts = copyXiangqiPlyFacts(xiangqiPlyFacts)
	g.State.ShogiPositionCounts = copyStringIntMap(shogiPositionCounts)
	g.State.ShogiPositionKeys = append([]string(nil), shogiPositionKeys...)
	g.State.ShogiPlyFacts = copyShogiPlyFacts(shogiPlyFacts)
}

// copyStringIntMap - returns copy string int map
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

// copyXiangqiPlyFacts - copies xiangqi ply facts
func copyXiangqiPlyFacts(in []xiangqiPlyFact) []xiangqiPlyFact {
	if in == nil {
		return nil
	}
	out := make([]xiangqiPlyFact, len(in))
	for i, fact := range in {
		out[i] = xiangqiPlyFact{
			Side:      fact.Side,
			GaveCheck: fact.GaveCheck,
			ChaseKeys: append([]string(nil), fact.ChaseKeys...),
		}
	}
	return out
}

// copyShogiPlyFacts - copies shogi ply facts
func copyShogiPlyFacts(in []shogiPlyFact) []shogiPlyFact {
	if in == nil {
		return nil
	}
	out := make([]shogiPlyFact, len(in))
	copy(out, in)
	return out
}
