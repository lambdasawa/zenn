package parser

import (
	"strings"

	"github.com/olivere/elastic"
)

func parseOr(input string) elastic.Query {
	orInputs := strings.Split(input, " OR ")

	queries := make([]elastic.Query, len(orInputs))
	for i, input := range orInputs {
		queries[i] = parseAnd(input)
	}

	return elastic.NewBoolQuery().Should(queries...)
}

func parseAnd(input string) elastic.Query {
	andInputs := strings.Split(input, " AND ")

	queries := make([]elastic.Query, len(andInputs))
	for i, input := range andInputs {
		queries[i] = parseSimple(input)
	}

	return elastic.NewBoolQuery().Filter(queries...)
}

func parseSimple(input string) elastic.Query {
	pair := strings.Split(input, ":")
	key, value := pair[0], pair[1]

	switch key {
	case "since":
		return elastic.NewRangeQuery("createdAt").From(value)
	case "until":
		return elastic.NewRangeQuery("createdAt").To(value)
	default:
		panic("unknown key")
	}
}
