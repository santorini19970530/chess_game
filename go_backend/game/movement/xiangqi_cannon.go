package movement

// XiangqiCannonStrategy — slides like a chariot; captures only by jumping exactly one screen.
type XiangqiCannonStrategy struct{}

func (XiangqiCannonStrategy) Name() string { return "XiangqiCannon" }

func (XiangqiCannonStrategy) LegalMoves(board any, from any) []any {
	ctx, ok := board.(MovementBoard)
	if !ok {
		return nil
	}
	src, ok := from.(Square)
	if !ok {
		return nil
	}
	directions := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	legal := make([]any, 0, 17)
	for _, dir := range directions {
		file, rank := src.File+dir[0], src.Rank+dir[1]
		jumped := false
		for isInsideXiangqiBoard(file, rank) {
			target, occupied := getPieceAt(file, rank)
			if !jumped {
				if !occupied {
					legal = append(legal, Square{File: file, Rank: rank})
				} else {
					jumped = true
				}
			} else {
				if occupied {
					if target.Color != ctx.Color {
						legal = append(legal, Square{File: file, Rank: rank})
					}
					break
				}
			}
			file += dir[0]
			rank += dir[1]
		}
	}
	return legal
}
