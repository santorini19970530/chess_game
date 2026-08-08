// CM3070 FP code
// request_config.go - form helpers for mode, skill level, and clock

package handlers

import (
	"fmt"
	sessionpkg "go_backend/game/session"
	"net/http"
	"strconv"
	"strings"
)

// readGameConfigFromRequest - reads game configuration settings from the request
func readGameConfigFromRequest(r *http.Request) (sessionpkg.GameMode, sessionpkg.GameType, string, int, string, string, error) {
	mode := sessionpkg.GameMode(strings.TrimSpace(r.FormValue("mode")))
	if mode == "" {
		mode = sessionpkg.GameModeHumanVsHuman
	}
	gameType := sessionpkg.GameType(strings.TrimSpace(r.FormValue("type")))
	if gameType == "" {
		gameType = sessionpkg.GameTypeChess
	}
	humanColor := strings.TrimSpace(r.FormValue("humanColor"))
	if humanColor == "" {
		humanColor = "white"
	}
	fen := strings.TrimSpace(r.FormValue("fen"))
	aiGameCount := 1
	if raw := strings.TrimSpace(r.FormValue("aiGameCount")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return "", "", "", 0, "", "", fmt.Errorf("invalid ai game count")
		}
		aiGameCount = parsed
	}
	profile := strings.TrimSpace(r.FormValue("aiProfile"))
	if profile == "" {
		profile = strings.TrimSpace(r.FormValue("profile")) // fallback name
	}
	return mode, gameType, humanColor, aiGameCount, fen, profile, nil
}

// readSkillLevelFromRequest - reads skill level settings from the request
func readSkillLevelFromRequest(r *http.Request) string {
	skillLevel := strings.TrimSpace(r.FormValue("skillLevel"))
	if skillLevel == "" {
		skillLevel = strings.TrimSpace(r.FormValue("coachLevel"))
	}
	return skillLevel
}

// applySkillLevelFromRequest - applies skill level settings from the request to the game session
func applySkillLevelFromRequest(gameID string, r *http.Request, game sessionpkg.GameSession) sessionpkg.GameSession {
	skillLevel := readSkillLevelFromRequest(r)
	if skillLevel == "" {
		return game
	}
	updated, err := sessionpkg.SetSkillLevelByID(gameID, skillLevel)
	if err != nil {
		return game
	}
	return updated
}

// formHasClockFields - reports whether the request form includes any clock fields
func formHasClockFields(r *http.Request) bool {
	for _, key := range []string{
		"clockEnabled",
		"whiteInitialMs", "blackInitialMs",
		"humanInitialMs", "aiInitialMs",
		"incrementMs",
	} {
		if strings.TrimSpace(r.FormValue(key)) != "" {
			return true
		}
	}
	return false
}

// parseOptionalNonNegInt64 - parses an optional non-negative integer from a string
func parseOptionalNonNegInt64(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v < 0 {
		return 0, fmt.Errorf("invalid clock value %q", raw)
	}
	return v, nil
}

// readClockFromRequest - parses optional clock fields from the request; omitted means leave the session clock alone
func readClockFromRequest(r *http.Request, humanColor string) (whiteMs, blackMs, incrementMs int64, present bool, err error) {
	if !formHasClockFields(r) {
		return 0, 0, 0, false, nil
	}
	present = true
	switch strings.ToLower(strings.TrimSpace(r.FormValue("clockEnabled"))) {
	case "0", "false", "off", "no":
		return 0, 0, 0, true, nil
	}

	whiteMs, err = parseOptionalNonNegInt64(r.FormValue("whiteInitialMs"))
	if err != nil {
		return 0, 0, 0, true, err
	}
	blackMs, err = parseOptionalNonNegInt64(r.FormValue("blackInitialMs"))
	if err != nil {
		return 0, 0, 0, true, err
	}
	humanMs, err := parseOptionalNonNegInt64(r.FormValue("humanInitialMs"))
	if err != nil {
		return 0, 0, 0, true, err
	}
	aiMs, err := parseOptionalNonNegInt64(r.FormValue("aiInitialMs"))
	if err != nil {
		return 0, 0, 0, true, err
	}
	if strings.TrimSpace(r.FormValue("humanInitialMs")) != "" || strings.TrimSpace(r.FormValue("aiInitialMs")) != "" {
		whiteMs, blackMs = sessionpkg.ClockSidesFromHumanAI(humanColor, humanMs, aiMs)
	}
	incrementMs, err = parseOptionalNonNegInt64(r.FormValue("incrementMs"))
	if err != nil {
		return 0, 0, 0, true, err
	}
	return whiteMs, blackMs, incrementMs, true, nil
}

// applyClockFromRequest - applies clock settings from the request to the game session
func applyClockFromRequest(gameID string, r *http.Request, humanColor string, game sessionpkg.GameSession) (sessionpkg.GameSession, error) {
	whiteMs, blackMs, incrementMs, present, err := readClockFromRequest(r, humanColor)
	if err != nil {
		return game, err
	}
	if !present {
		return game, nil
	}
	updated, err := sessionpkg.SetClockByID(gameID, whiteMs, blackMs, incrementMs)
	if err != nil {
		return game, err
	}
	return updated, nil
}
