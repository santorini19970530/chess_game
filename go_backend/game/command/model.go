// CM3070 FP code
// model.go - command model types for move parsing

package command

// parsedCommand is a lightweight parsed command model
type ParsedCommand struct {
	Raw       string
	Normalized string
	Format    string
	PieceCode string
	FromFile  byte
	FromRank  int
	ToFile    byte
	ToRank    int
	Promotion string
}
