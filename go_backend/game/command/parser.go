package command

import (
	"fmt"
	"log"
	"regexp"
)

var commandFormatPattern = regexp.MustCompile(`^(?:[a-h][1-8][a-h][1-8][qrbn]?|[prnbqk][a-h][1-8][a-h][1-8])$`)

func ParseCommand(command string) (ParsedCommand, error) {
	if !commandFormatPattern.MatchString(command) {
		return ParsedCommand{}, fmt.Errorf("invalid command format")
	}

	parsed := ParsedCommand{
		Raw:    command,
		Format: "uci",
	}

	if command[1] >= '1' && command[1] <= '8' {
		parsed.FromFile = command[0]
		parsed.FromRank = int(command[1] - '0')
		parsed.ToFile = command[2]
		parsed.ToRank = int(command[3] - '0')
		if len(command) == 5 {
			parsed.Promotion = string(command[4])
		}
	} else {
		parsed.Format = "piece-prefixed"
		parsed.PieceCode = string(command[0])
		parsed.FromFile = command[1]
		parsed.FromRank = int(command[2] - '0')
		parsed.ToFile = command[3]
		parsed.ToRank = int(command[4] - '0')
	}

	return parsed, nil
}

func ParseAndLogCommand(command string) error {
	parsed, err := ParseCommand(command)
	if err != nil {
		return err
	}

	fromFileInt := int(parsed.FromFile - 'a' + 1)
	toFileInt := int(parsed.ToFile - 'a' + 1)

	log.Printf(
		"command parsed: raw=%q format=%s piece=%q from=%c%d(file=%d,rank=%d) to=%c%d(file=%d,rank=%d) promotion=%q",
		parsed.Raw,
		parsed.Format,
		parsed.PieceCode,
		parsed.FromFile,
		parsed.FromRank,
		fromFileInt,
		parsed.FromRank,
		parsed.ToFile,
		parsed.ToRank,
		toFileInt,
		parsed.ToRank,
		parsed.Promotion,
	)

	return nil
}
