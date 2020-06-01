package models

type OpponentsDefeated struct {
	RankAtt    int `json:"rankAtt" pg:",use_zero" gqlgen:"rankAtt"`
	ScoreAtt   int `pg:",use_zero" json:"scoreAtt" gqlgen:"scoreAtt"`
	RankDef    int `pg:",use_zero" json:"rankDef" gqlgen:"rankDef"`
	ScoreDef   int `pg:",use_zero" json:"scoreDef" gqlgen:"scoreDef"`
	RankTotal  int `pg:",use_zero" json:"rankTotal" gqlgen:"rankTotal"`
	ScoreTotal int `pg:",use_zero" json:"scoreTotal" gqlgen:"scoreTotal"`
	RankSup    int `pg:",use_zero" json:"rankSup" gqlgen:"rankSup"`
	ScoreSup   int `pg:",use_zero" json:"scoreSup" gqlgen:"scoreSup"`
}

type OpponentsDefeatedFilter struct {
	RankAtt       int `json:"rankAtt" gqlgen:"rankAtt"`
	RankAttGT     int `json:"rankAttGT" gqlgen:"rankAttGT"`
	RankAttGTE    int `json:"rankAttGTE" gqlgen:"rankAttGTE"`
	RankAttLT     int `json:"rankAttLT" gqlgen:"rankAttLT"`
	RankAttLTE    int `json:"rankAttLTE" gqlgen:"rankAttLTE"`
	ScoreAtt      int `json:"scoreAtt" gqlgen:"scoreAtt"`
	ScoreAttGT    int `json:"scoreAttGT" gqlgen:"scoreAttGT"`
	ScoreAttGTE   int `json:"scoreAttGTE" gqlgen:"scoreAttGTE"`
	ScoreAttLT    int `json:"scoreAttLT" gqlgen:"scoreAttLT"`
	ScoreAttLTE   int `json:"scoreAttLTE" gqlgen:"scoreAttLTE"`
	RankDef       int `json:"rankDef" gqlgen:"rankDef"`
	RankDefGT     int `json:"rankDefGT" gqlgen:"rankDefGT"`
	RankDefGTE    int `json:"rankDefGTE" gqlgen:"rankDefGTE"`
	RankDefLT     int `json:"rankDefLT" gqlgen:"rankDefLT"`
	RankDefLTE    int `json:"rankDefLTE" gqlgen:"rankDefLTE"`
	ScoreDef      int `json:"scoreDef" gqlgen:"scoreDef"`
	ScoreDefGT    int `json:"scoreDefGT" gqlgen:"scoreDefGT"`
	ScoreDefGTE   int `json:"scoreDefGTE" gqlgen:"scoreDefGTE"`
	ScoreDefLT    int `json:"scoreDefLT" gqlgen:"scoreDefLT"`
	ScoreDefLTE   int `json:"scoreDefLTE" gqlgen:"scoreDefLTE"`
	RankTotal     int `json:"rankTotal" gqlgen:"rankTotal"`
	RankTotalGT   int `json:"rankTotalGT" gqlgen:"rankTotalGT"`
	RankTotalGTE  int `json:"rankTotalGTE" gqlgen:"rankTotalGTE"`
	RankTotalLT   int `json:"rankTotalLT" gqlgen:"rankTotalLT"`
	RankTotalLTE  int `json:"rankTotalLTE" gqlgen:"rankTotalLTE"`
	ScoreTotal    int `json:"scoreTotal" gqlgen:"scoreTotal"`
	ScoreTotalGT  int `json:"scoreTotalGT" gqlgen:"scoreTotalGT"`
	ScoreTotalGTE int `json:"scoreTotalGTE" gqlgen:"scoreTotalGTE"`
	ScoreTotalLT  int `json:"scoreTotalLT" gqlgen:"scoreTotalLT"`
	ScoreTotalLTE int `json:"scoreTotalLTE" gqlgen:"scoreTotalLTE"`
	ScoreSup      int `json:"scoreSup" gqlgen:"scoreSup"`
	ScoreSupGT    int `json:"scoreSupGT" gqlgen:"scoreSupGT"`
	ScoreSupGTE   int `json:"scoreSupGTE" gqlgen:"scoreSupGTE"`
	ScoreSupLT    int `json:"scoreSupLT" gqlgen:"scoreSupLT"`
	ScoreSupLTE   int `json:"scoreSupLTE" gqlgen:"scoreSupLTE"`
	RankSup       int `json:"rankSup" gqlgen:"rankSup"`
	RankSupGT     int `json:"rankSupGT" gqlgen:"rankSupGT"`
	RankSupGTE    int `json:"rankSupGTE" gqlgen:"rankSupGTE"`
	RankSupLT     int `json:"rankSupLT" gqlgen:"rankSupLT"`
	RankSupLTE    int `json:"rankSupLTE" gqlgen:"rankSupLTE"`
}
