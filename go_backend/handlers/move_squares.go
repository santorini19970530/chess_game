package handlers

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	commandpkg "go_backend/game/command"
	pieces "go_backend/game/piece"
	sessionpkg "go_backend/game/session"
)

// Fairy / Xiangqi / Shogi UCI: files a-i, ranks 1-10 (optional trailing '+' for shogi promote).
var variantUCIPattern = regexp.MustCompile(`^([a-i])([0-9]{1,2})([a-i])([0-9]{1,2})\+?$`)

// Shogi drop: P*e5 / p@e5 (piece letter + *|@ + square). from is empty.
var variantDropPattern = regexp.MustCompile(`^[plnsgbr][*@]([a-i])([1-9])$`)

func parseVariantUCISquares(move string) (fromFile string, fromRank int, toFile string, toRank int, err error) {
	move = strings.ToLower(strings.TrimSpace(move))
	m := variantUCIPattern.FindStringSubmatch(move)
	if m == nil {
		return "", 0, "", 0, fmt.Errorf("invalid move %q", move)
	}
	fromRank, err = strconv.Atoi(m[2])
	if err != nil {
		return "", 0, "", 0, err
	}
	toRank, err = strconv.Atoi(m[4])
	if err != nil {
		return "", 0, "", 0, err
	}
	return m[1], fromRank, m[3], toRank, nil
}

func parseVariantDropSquares(move string) (toFile string, toRank int, ok bool) {
	move = strings.ToLower(strings.TrimSpace(move))
	m := variantDropPattern.FindStringSubmatch(move)
	if m == nil {
		return "", 0, false
	}
	rank, err := strconv.Atoi(m[2])
	if err != nil {
		return "", 0, false
	}
	return m[1], rank, true
}

// resolveMoveSquares validates command shape before ApplyMove.
// Chess keeps the a-h/1-8 (+ SAN) parser; Xiangqi/Shogi skip it (file i / rank 10).
func resolveMoveSquares(
	gameType sessionpkg.GameType, commandText string, expectedColor pieces.PieceColor,
) (fromFile string, fromRank int, toFile string, toRank int, err error) {
	switch gameType {
	case sessionpkg.GameTypeXiangqi, sessionpkg.GameTypeShogi:
		if gameType == sessionpkg.GameTypeShogi {
			if tf, tr, ok := parseVariantDropSquares(commandText); ok {
				log.Printf("command parsed: raw=%q format=drop game=shogi to=%s%d", commandText, tf, tr)
				return "", 0, tf, tr, nil
			}
		}
		fromFile, fromRank, toFile, toRank, err = parseVariantUCISquares(commandText)
		if err != nil {
			return "", 0, "", 0, err
		}
		log.Printf("command parsed: raw=%q format=uci game=%s from=%s%d to=%s%d",
			commandText, gameType, fromFile, fromRank, toFile, toRank)
		return fromFile, fromRank, toFile, toRank, nil
	default:
		parsed, err := commandpkg.ParseCommandForColor(commandText, expectedColor)
		if err != nil {
			return "", 0, "", 0, err
		}
		if err := commandpkg.ParseAndLogCommandForColor(commandText, expectedColor); err != nil {
			return "", 0, "", 0, err
		}
		return string(parsed.FromFile), parsed.FromRank, string(parsed.ToFile), parsed.ToRank, nil
	}
}
