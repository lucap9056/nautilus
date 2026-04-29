package builtins

import (
	"encoding/csv"
	"net/http"
	"strings"
)

type Factory func(args ...string) http.HandlerFunc

func ParseArguments(input string) ([]string, error) {
	r := csv.NewReader(strings.NewReader(input))

	r.LazyQuotes = true

	r.TrimLeadingSpace = true

	fields, err := r.Read()
	if err != nil {
		return nil, err
	}
	return fields, nil
}

func ParseDirective(s string) (string, []string, error) {
	var args []string
	start := strings.Index(s, "(")
	end := strings.LastIndex(s, ")")
	if start != -1 && end != -1 && end > start {
		fields, err := ParseArguments(s[start+1 : end])
		if err != nil {
			return "", nil, err
		}
		s = s[:start]
		args = fields
	}
	return s, args, nil
}
