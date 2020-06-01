package models

type ServerStatus string

const (
	ServerStatusOpen  = "open"
	ServerStatusClose = "close"
)

type Server struct {
	tableName struct{} `pg:"alias:server"`

	ID     int          `json:"id" gqlgen:"id"`
	Key    string       `json:"key" gqlgen:"key" pg:",unique"`
	Status ServerStatus `json:"status" gqlgen:"status"`

	LangVersionTag LanguageTag  `json:"langVersionTag" gqlgen:"langVersionTag"`
	LangVersion    *LangVersion `json:"langVersion,omitempty" gqlgen:"-"`
}

type ServerFilter struct {
	tableName struct{} `urlstruct:"server"`

	ID    []int `json:"id" gqlgen:"id"`
	IdNEQ []int `json:"idNEQ" gqlgen:"idNEQ"`

	Key      []string `json:"key" gqlgen:"key"`
	KeyNEQ   []string `json:"keyNEQ" gqlgen:"keyNEQ"`
	KeyMATCH string   `json:"keyMATCH" gqlgen:"keyMATCH"`
	KeyIEQ   string   `json:"keyIEQ" gqlgen:"keyIEQ"`

	Status    []string `json:"status" gqlgen:"status"`
	StatusNIN []string `json:"statusNIN" gqlgen:"statusNIN"`

	LangVersionTag    []string `json:"langVersionTag" gqlgen:"langVersionTag"`
	LangVersionTagNIN []string `json:"langVersionTagNIN" gqlgen:"langVersionTagNIN"`

	Offset int    `urlstruct:",nowhere" json:"offset" gqlgen:"offset"`
	Limit  int    `urlstruct:",nowhere" json:"limit" gqlgen:"limit"`
	Sort   string `urlstruct:",nowhere" json:"sort" gqlgen:"sort"`
}
