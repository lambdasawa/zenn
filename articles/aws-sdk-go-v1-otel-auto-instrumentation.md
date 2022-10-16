---
title: "AWS SDK for Go v1 を使った API コールに対して OpenTelemetry の Span を自動で作成する"
emoji: "👁️"
type: "tech" # tech: 技術記事 / idea: アイデア
topics: ["go", "opentelemetry"]
published: false
---

## はじめに

S3, SQS, DynamoDB など AWS のサービスを活用した Go アプリケーションを運用する際、 OpenTelemetry などを使って分散トレーシングを導入したくなることがあると思います。

[otelaws](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws) としてそのためのライブラリが公開されており、基本的にはこれを導入するだけで対応できます。
しかし、これは AWS SDK for Go v2 に対応しており、 v1 には対応していません。

この記事では v2 にバージョンアップできない場合などに使える代替案を紹介します。

まず v2 ではどのように対応されているのか、コードを読んで確認します。
以下のファイルを読み込めば大体の内容がわかります。

<https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/aws.go>

v2 では `AppendMiddlewares` という関数をクライアントオブジェクトに対して呼び出してあげれば自動で Span が作成されるようです。

ところで AWS SDK for Go の middleware と v2 で導入された仕組みです。 ([AWS の発表](https://aws.amazon.com/jp/about-aws/whats-new/2021/01/aws-sdk-for-go-version-2-now-generally-available/))
v1 には middleware は存在しないので、このコードをそのまま利用するのは難しそうです。

しかし v1 には `Handlers` という構造体があり、これが middleware のような役割を果たしています
詳しくは [builders.flash](https://aws.amazon.com/jp/builders-flash/202206/backstage-aws-sdk-02/?awsf.filter-name=*all) の記事が参考になります。

この記事ではこの `Handlers` を使用して、自動で Span を作る仕組みを紹介します。

## 実装

ここでは Exporter として [Jaeger](https://www.jaegertracing.io/) を使うことにします。
また、実際の AWS 環境にはリクエストは飛ばさず [LocalStack](https://github.com/localstack/localstack) を使って動作確認することにします。

これらを起動する Docker Compose の設定は以下のようになります。

```yml
version: '3.8'
services:
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "14268:14268" # for exporter
      - "16686:16686" # for browser
  localstack:
    image: localstack/localstack
    ports:
      - "127.0.0.1:4566:4566"
```

`Handlers` を用いて自動で `Span` を作成するための関数は以下のようになります。

```go
import "github.com/aws/aws-sdk-go/aws/client"

func instrument(client *client.Client) {
	serviceName := client.ServiceName

	client.Handlers.Send.PushFront(func(r *request.Request) {
		operationName := r.Operation.Name

		ctx, _ := tracer.Start(r.Context(), fmt.Sprintf("%s:%s", serviceName, operationName))

		r.SetContext(ctx)
	})

	client.Handlers.Complete.PushBack(func(r *request.Request) {
		span := trace.SpanFromContext(r.Context())
		defer span.End()

		if r.Error != nil {
			span.SetStatus(codes.Error, r.Error.Error())
			span.RecordError(r.Error)
		}
	})
}
```

実際の `otelaws` は timestamp, span kind, 複数の attribute を設定していますが、ここでは省略しています。
ここは v2 でも v1 でも対して変わらないので、以下のコードを参考にすることによって自分で簡単に設定できるのではないかと思います。

<https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws/aws.go>

先ほど作成した関数は以下のように呼び出すことができます。

```go
sess := session.Must(session.NewSession(&aws.Config{
	Endpoint:         aws.String("http://localhost:4566"),
	Region:           aws.String("us-east-1"),
	S3ForcePathStyle: aws.Bool(true),
}))

{
	service := s3.New(sess)
	instrument(service.Client) // ここで Handlers にフックが設定される

	_, _ = service.ListBucketsWithContext(ctx, &s3.ListBucketsInput{})
}

{
	service := dynamodb.New(sess) // ここで Handlers にフックが設定される
	instrument(service.Client)

	_, _ = service.ListTablesWithContext(ctx, &dynamodb.ListTablesInput{})
	_, _ = service.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("not-found"),
		Item:      map[string]*dynamodb.AttributeValue{"id": {S: aws.String("1")}},
	})
}
```

このコードを呼び出した結果は Jaeger で見ると以下のようになります。

![](https://storage.googleapis.com/zenn-user-upload/d6c1455e6d88-20221016.png)
