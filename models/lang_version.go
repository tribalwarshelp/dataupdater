package models

type LanguageTag string

type LangVersion struct {
	Tag      LanguageTag `pg:",pk" json:"tag" gqlgen:"tag"`
	Name     string      `json:"name" gqlgen:"name" pg:",unique"`
	Host     string      `json:"host" gqlgen:"host"`
	Timezone string      `json:"timezone" gqlgen:"timezone"`
}
