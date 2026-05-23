package command

// ParsedCommand is a lightweight parsed command model.
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
