package graph

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// NormalizeName is the authoritative equality transform for Tag and alias
// names. It deliberately does not generate a slug or infer a classification.
func NormalizeName(value string) string {
	value = norm.NFKC.String(value)
	value = cases.Fold().String(value)
	return strings.Join(strings.Fields(value), " ")
}
