package parser

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/olivere/elastic"
	"github.com/stretchr/testify/require"
)

func serialize(q elastic.Query) string {
	v, _ := q.Source()
	buf := new(bytes.Buffer)
	_ = json.NewEncoder(buf).Encode(v)
	return buf.String()
}

func Test_praseSimple(t *testing.T) {
	query := parseSimple("until:2021-12-31")

	const expected = `{
  "range": {
    "createdAt": {
      "from": null,
      "include_lower": true,
      "include_upper": true,
      "to": "2021-12-31"
    }
  }
}`

	actual := serialize(query)

	require.JSONEq(t, expected, actual)
}

func Test_parseOr(t *testing.T) {
	query := parseOr("since:2021-01-01 AND until:2021-12-31 OR since:1921-01-01 AND until:1921-12-31")

	const expected = `{
  "bool": {
    "should": [
      {
        "bool": {
          "filter": [
            {
              "range": {
                "createdAt": {
                  "from": "2021-01-01",
                  "include_lower": true,
                  "include_upper": true,
                  "to": null
                }
              }
            },
            {
              "range": {
                "createdAt": {
                  "from": null,
                  "include_lower": true,
                  "include_upper": true,
                  "to": "2021-12-31"
                }
              }
            }
          ]
        }
      },
      {
        "bool": {
          "filter": [
            {
              "range": {
                "createdAt": {
                  "from": "1921-01-01",
                  "include_lower": true,
                  "include_upper": true,
                  "to": null
                }
              }
            },
            {
              "range": {
                "createdAt": {
                  "from": null,
                  "include_lower": true,
                  "include_upper": true,
                  "to": "1921-12-31"
                }
              }
            }
          ]
        }
      }
    ]
  }
}`

	actual := serialize(query)

	require.JSONEq(t, expected, actual)
}
