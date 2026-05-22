package command

import (
	"fmt"
	"log"
	"regexp"
)

var commandFormatPattern = regexp.MustCompile(`^(?:[a-h][1-8][a-h][1-8][qrbn]?|[prnbqk][a-h][1-8][a-h][1-8])$`)

func ParseAndLogCommand(command string) error {
	if !commandFormatPattern.MatchString(command) {
		return fmt.Errorf("invalid command format")
	}

	format := "uci"
	pieceCode := ""
	promotion := ""

	var fromFile byte
	var fromRank byte
	var toFile byte
	var toRank byte

	if command[1] >= '1' && command[1] <= '8' {
		fromFile = command[0]
		fromRank = command[1]
		toFile = command[2]
		toRank = command[3]
		if len(command) == 5 {
			promotion = string(command[4])
		}
	} else {
		format = "piece-prefixed"
		pieceCode = string(command[0])
		fromFile = command[1]
		fromRank = command[2]
		toFile = command[3]
		toRank = command[4]
	}

	fromFileInt := int(fromFile - 'a' + 1)
	fromRankInt := int(fromRank - '0')
	toFileInt := int(toFile - 'a' + 1)
	toRankInt := int(toRank - '0')

	log.Printf(
		"command parsed: raw=%q format=%s piece=%q from=%c%d(file=%d,rank=%d) to=%c%d(file=%d,rank=%d) promotion=%q",
		command,
		format,
		pieceCode,
		fromFile,
		fromRankInt,
		fromFileInt,
		fromRankInt,
		toFile,
		toRankInt,
		toFileInt,
		toRankInt,
		promotion,
	)

	return nil
}
