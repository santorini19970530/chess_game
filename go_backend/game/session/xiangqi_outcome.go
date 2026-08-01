// CM3070 FP code
// xiangqi_outcome.go - dispatches outcome evaluation by game type

package session

// evaluateOutcomeForGameType - dispatches outcome evaluation by game type
func evaluateOutcomeForGameType(gameType GameType) GameOutcome {
	switch gameType {
	case GameTypeXiangqi:
		return EvaluateXiangqiGameOutcome()
	case GameTypeShogi:
		return EvaluateShogiGameOutcome()
	default:
		return EvaluateGameOutcome()
	}
}
