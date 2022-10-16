package main

import (
	"bytes"
	"encoding/json"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/olivere/elastic"
)

var (
	lex = lexer.MustSimple([]lexer.SimpleRule{
		{Name: "Key", Pattern: `[a-zA-Z_]+`},
		{Name: "Value", Pattern: `[a-zA-Z0-9_-]+`},
		{Name: "Punct", Pattern: `[():]`},
		{Name: "whitespace", Pattern: `\s+`},
	})

	parser = participle.MustBuild[RootQuery](participle.Lexer(lex))
)

type (
	RootQuery = OrQuery

	OrQuery struct {
		Head AndQuery `@@`
		Tail []OrTail `@@*`
	}

	OrTail struct {
		Separator struct{} `'OR'`
		Tail      AndQuery `@@*`
	}

	AndQuery struct {
		Head SimpleOrGroup `@@`
		Tail []AndTail     `@@*`
	}

	AndTail struct {
		Separator struct{}      `'AND'`
		Tail      SimpleOrGroup `@@*`
	}

	SimpleOrGroup struct {
		SimpleQuery *SimpleQuery `@@`
		GroupQuery  *RootQuery   `| '(' @@ ')'`
	}

	SimpleQuery struct {
		Key       string   `@Key`
		Separator struct{} `':'`
		Value     string   `@Value`
	}
)

func (o *OrQuery) ToElastic() elastic.Query {
	queries := make([]elastic.Query, 0)

	queries = append(queries, o.Head.ToElastic())

	for _, tail := range o.Tail {
		queries = append(queries, tail.Tail.ToElastic())
	}

	return elastic.NewBoolQuery().Should(queries...)
}

func (o *AndQuery) ToElastic() elastic.Query {
	queries := make([]elastic.Query, 0)

	queries = append(queries, o.Head.ToElastic())

	for _, tail := range o.Tail {
		queries = append(queries, tail.Tail.ToElastic())
	}

	return elastic.NewBoolQuery().Filter(queries...)
}

func (o *SimpleOrGroup) ToElastic() elastic.Query {
	if o.SimpleQuery != nil {
		return o.SimpleQuery.ToElastic()
	}

	if o.GroupQuery != nil {
		return o.GroupQuery.ToElastic()
	}

	panic("invalid tree")
}

func (o *SimpleQuery) ToElastic() elastic.Query {
	switch o.Key {
	case "since":
		return elastic.NewRangeQuery("createdAt").From(o.Value)
	case "until":
		return elastic.NewRangeQuery("createdAt").To(o.Value)
	default:
		panic("unknown key")
	}
}

func main() {
	rootQuery, err := parser.ParseString("", "since:2021 AND (since:2023 OR until:2024) AND since:2025")
	if err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	src, _ := rootQuery.ToElastic().Source()
	_ = json.NewEncoder(buf).Encode(src)
	println(buf.String())
}
