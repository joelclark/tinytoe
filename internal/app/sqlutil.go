package app

import "strings"

func quoteIdent(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

func qualifyIdent(schema, name string) string {
	return quoteIdent(schema) + "." + quoteIdent(name)
}
