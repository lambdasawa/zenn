---
title: "Goで複雑な検索クエリをパースする"
emoji: "🅿️"
type: "tech" # tech: 技術記事 / idea: アイデア
topics: ["go"]
published: true
---

## はじめに

Go で複雑な検索クエリを処理する方法の1例を紹介します。

以下のようなシチュエーションを想定します。

- 私達が Twitter のような Web サービスを運用開発しているとします。
- 単なる本文からの検索以外にもサービスが独自で定めた構文のクエリをサポートすることにしたいです。
- この構文は以下のような仕様にしたいです。
  - `until:2021-12-31` というように、キーワードに続けて `:` で区切って値を指定することで条件を指定できる (仕様1)
  - `since:2021-01-01 AND until:2021-12-31` というように、 各条件は `AND`, `OR` というキーワードを使って連結できる。一般的なプログラミング言語と同様、 `OR` より `AND` を先に評価する。 (仕様2)
  - `until:2021-12-31 AND (from:emacs_user OR from:vim_user)` というように、 `()` で囲むことで条件をグループ化できる。`a OR (b AND (c OR (d AND e)))` のように、一般的なプログラミング言語と同様にネストすることがある。 (仕様3)
- サービスのバックエンドは Go と Elasticsearch を使っているとします。
  - Elasticsearch の API クライアントとして [github.com/olivere/elastic](https://github.com/olivere/elastic) を使っている
  - この記事の本質とは無関係ですが、サンプルコードで使うため明記しました

## 対応方針

仕様1について考えてみます。
これは単に `:` でスプリットすればよいだけでしょう。以下のようなコードで対応できると思います。

```go
const input = "until:2021-12-31"

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
```

仕様2を考えてみます。
優先順位の問題がありますね。最初に優先順位の低い `OR` でスプリットして、スプリットされた文字列それぞれに対して `AND` でスプリットするとよさそうです。

```go
const input = "from:2021-01-01 AND until:2021-12-31 OR from:1921-01-01 AND until:1921-12-31"

func parseOr(input string) elastic.Query {
	orInputs := strings.Split(input, " OR ")

	queries := make([]elastic.Query, len(orInputs))
	for i, input := range orInputs {
		queries[i] = parseAnd(input)
	}

	// elasticsearch では should で OR 検索になる
	return elastic.NewBoolQuery().Should(queries...)
}

func parseAnd(input string) elastic.Query {
	andInputs := strings.Split(input, " AND ")

	queries := make([]elastic.Query, len(andInputs))
	for i, input := range andInputs {
		queries[i] = parseSimple(input)
	}

	// elasticsearch では filter で AND 検索になる
	return elastic.NewBoolQuery().Filter(queries...)
}
```

仕様3を考えてみます。
`()` で囲まれた部分を再帰的にパースする必要があります。手書きで文字列処理をするとして、そこそこ複雑なアルゴリズムが必要になりそうです。正規表現だけでやるのも無理 or 大変そうです。

パーサジェネレータとかパーサコンビネータと呼ばれるライブラリ/フレームワークを使うと、比較的簡単にこのような複雑な文字列を構造化された値に落とし込むことができます。

今回は [github.com/alecthomas/participle](https://github.com/alecthomas/participle) を使ってこの仕様を満たす実装を作ってみます。
`README.md` と `_examples/` が充実しているので、パーサの実装に慣れている方はそちらを読んでいただくのが手っ取り早いと思います。

## 実装

まず `AND` も `OR` も `()` も存在しない簡単なケースから考えます。

```go
var (
	// 字句解析器
	lex = lexer.MustSimple([]lexer.SimpleRule{
		{Name: "Key", Pattern: `[a-zA-Z_]+`}, // コロンの左側を表す正規表現
		{Name: "Value", Pattern: `[a-zA-Z0-9_-]+`}, // コロンの右側を表す正規表現
		{Name: "Punct", Pattern: `[():]`},
		{Name: "whitespace", Pattern: `\s+`},
	})

	// 構文解析器
	// これを使うと単なる string を RootQuery 型の構造体に変換できる
	parser = participle.MustBuild[RootQuery](participle.Lexer(lex))
)

type (
	RootQuery = SimpleQuery

	SimpleQuery struct {
		// @Key は字句解析器で定義した Key にマッチさせるという意味
		Key       string   `@Key`

		// コロンの1文字にマッチさせるという意味
		Separator struct{} `':'`

		// @Value は字句解析器で定義した Value にマッチさせるという意味
		Value     string   `@Value`
	}
)

func main() {
	rootQuery, _ := parser.ParseString("", `until:2021-12-31`)
	fmt.Printf("%+v\n", rootQuery)
}
```

次は `AND` が存在するケースを考えます。こんな感じの定義で対応できます。

```diff
type (
-	RootQuery = SimpleQuery
+	RootQuery = AndQuery

+	AndQuery struct {
+		// @@ を指定してこのフィールドの型の構造体の定義にマッチさせる
+		Head SimpleQuery `@@`
+
+		// アスタリスクは繰り返しを意味する
+		Tail []AndTail   `@@*`
+	}
+
+	AndTail struct {
+		Separator struct{}      `'AND'`
+		Tail      []SimpleQuery `@@*
+	}
+
	SimpleQuery struct {
		Key       string   `@Key`
		Separator struct{} `':'`
		Value     string   `@Value`
	}
)
```

加えて `OR` が存在するケースを考えます。こんな感じの定義で対応できます。

```diff
type (
-	RootQuery = AndQuery
+	RootQuery = OrQuery

+	OrQuery struct {
+		Head AndQuery `@@`
+		Tail []OrTail `@@*`
+	}
+
+	OrTail struct {
+		Separator struct{}   `'OR'`
+		Tail      []AndQuery `@@*`
+	}
+
	AndQuery struct {
		Head SimpleQuery `@@`
		Tail []AndTail   `@@*`
	}

	AndTail struct {
		Separator struct{}      `'AND'`
		Tail      []SimpleQuery `@@*`
	}

	SimpleQuery struct {
		Key       string   `@Key`
		Separator struct{} `':'`
		Value     string   `@Value`
	}
)
```

最後にこんな感じの定義を追加すると `()` をパースできるようになります。ネストしていても大丈夫です。

```diff
type (
	RootQuery = OrQuery

	OrQuery struct {
		Head AndQuery `@@`
		Tail []OrTail `@@*`
	}

	OrTail struct {
		Separator struct{}   `'OR'`
		Tail      []AndQuery `@@*`
	}

	AndQuery struct {
-		Head SimpleQuery `@@`
+		Head SimpleOrGroup `@@`
		Tail []AndTail     `@@*`
	}

	AndTail struct {
		Separator struct{}        `'AND'`
-		Tail      []SimpleQuery `@@*`
+		Tail      []SimpleOrGroup `@@*`
	}

+	SimpleOrGroup struct {
+		// パイプで分岐できる。これらは SimpleQuery か RootQuery のどちらかにマッチするという意味。
+		SimpleQuery *SimpleQuery `@@`
+		GroupQuery  *RootQuery   `| '(' @@ ')'`
+	}
+
	SimpleQuery struct {
		Key       string   `@Key`
		Separator struct{} `':'`
		Value     string   `@Value`
	}
)
```

ここまで定義できたら完璧な AST が作れたことになります。
あとは `RootQuery` を `elastic.Query` と 1:1 に対応させるロジックを書くことで全ての仕様を満たせます。

```go
func (o *OrQuery) ToElastic() elastic.Query {
	queries := make([]elastic.Query, 0)

	queries = append(queries, o.Head.ToElastic())

	for _, tail := range o.Tail {
		queries = append(queries, tail.Tail.ToElastic())
	}

	// elasticsearch では should で OR 検索になる
	return elastic.NewBoolQuery().Should(queries...) 
}

func (o *AndQuery) ToElastic() elastic.Query {
	queries := make([]elastic.Query, 0)

	queries = append(queries, o.Head.ToElastic())

	for _, tail := range o.Tail {
		queries = append(queries, tail.Tail.ToElastic())
	}

	// elasticsearch では filter で OR 検索になる
	return elastic.NewBoolQuery().Filter(queries...)
}

func (o *SimpleOrGroup) ToElastic() elastic.Query {
	if o.SimpleQuery != nil {
		return o.SimpleQuery.ToElastic()
	}

	if o.GroupQuery != nil {
		return o.GroupQuery.ToElastic()
	}

	panic("invalid tree") // サンプルなので適当に panic
}

func (o *SimpleQuery) ToElastic() elastic.Query {
	switch o.Key {
	case "since":
		return elastic.NewRangeQuery("createdAt").From(o.Value)
	case "until":
		return elastic.NewRangeQuery("createdAt").To(o.Value)
	default:
		panic("unknown key") // サンプルなので適当に panic
	}
}
```

あまり現実的ではないですが `since:2021 AND (since:2023 OR until:2024) AND since:2025` というクエリをこのパーサに食わせて `.ToElastic` を呼ぶと、以下のような Elasticsearch のクエリを生成できます。

```json
{
  "bool": {
    "should": {
      "bool": {
        "filter": [
          {
            "range": {
              "createdAt": { "from": "2021", "to": null }
            }
          },
          {
            "bool": {
              "should": [
                {
                  "bool": {
                    "filter": {
                      "range": {
                        "createdAt": { "from": "2023", "to": null }
                      }
                    }
                  }
                },
                {
                  "bool": {
                    "filter": {
                      "range": {
                        "createdAt": { "from": null, "to": "2024" }
                      }
                    }
                  }
                }
              ]
            }
          },
          {
            "range": {
              "createdAt": { "from": "2025", "to": null }
            }
          }
        ]
      }
    }
  }
}
```

`func (o *SimpleQuery) ToElastic()` の実装を追加するだけで、 `from` や `since` 以外にもイイね数やリツイート数、投稿者などで絞るクエリにも簡単に対応できると思います。

## 最後に

実行可能なコードは [GitHub](https://github.com/lambdasawa/zenn/blob/main/snippet/golang-parser-library-participle/by-generator/main.go) にあります。
必要に応じて参照してみてください。

一般的な Web サービスでこのようなクエリをサポートすることはあまり無いかもしれませんが、このような機能が充実しているとヘビーユーザはとても喜ぶと思います。
(個人的には Twitter のクエリをそこそこ活用しています。)

ニッチなケースかも知れませんが、このような機能の実装が必要になった際の参考になれば幸いです。
