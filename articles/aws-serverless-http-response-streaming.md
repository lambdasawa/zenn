---
title: "AWS 上でサーバレス構成で HTTP レスポンスをストリーミングする"
emoji: "🌌"
type: "tech" # tech: 技術記事 / idea: アイデア
topics: ["aws", "go", "http"]
publication_name: "microcms"
published: true
---

## はじめに

[AWS Lambda レスポンスストリーミングの紹介 | Amazon Web Services ブログ](https://aws.amazon.com/jp/blogs/news/introducing-aws-lambda-response-streaming/)

最近、上記のブログで Lambda でレスポンスをストリーミングできるようになったという話がありました。

自分がこのブログを読んだ時、ここで話されているストリーミングとは HTTP のレイヤで説明すると何者なのだろうか、という疑問を持ちました
最近この辺りを調査したので、その内容をこの記事でまとめます。

また、他の AWS + サーバレスな構成でストリーミングをする方法もついでに紹介します。

## 前提: サーバレスの種類について

この記事では以下のサービスについて考慮します。

- Lambda
- AppRunner
- API Gateway
- AppSync

Fargate がサーバレスの文脈で扱われることもあるかと思いますが、ECS と EKS は考慮しません。
手前に ALB があるなら EC2 と変わらないでしょうし、ローカルで動かしたものがそのまま AWS 上でも動作するのではないかと思います。

## 前提: ストリーミングの種類について

この記事ではストリーミングのやり方に関して以下の3種類を考慮します。

- Transfer-Encoding: chunked のレスポンスヘッダーを使う
  - [MDN の解説](https://developer.mozilla.org/ja/docs/Web/HTTP/Headers/Transfer-Encoding)
- Server Sent Event (SSE) を使う
  - [MDN の解説](https://developer.mozilla.org/ja/docs/Web/API/Server-sent_events/Using_server-sent_events)
- WebSocket を使う
  - [MDN の解説](https://developer.mozilla.org/ja/docs/Web/API/WebSockets_API)

## 前提: ローカルで動くサンプルコード

AWS を触る前にまずローカルで動くサンプルコードを作ります。
サーバとクライアントを両方作ります。
また、 `curl` でそのエンドポイントを叩くサンプルも示します。

サーバ実装のフレームワークとしては Go 言語の [echo](https://echo.labstack.com/guide/) を使います。
(ここで echo を選択した理由は自分の慣れであり、特にフレームワーク依存の話をするわけではありません)

クライアントは Vanilla JS です。

### Transfer-Encoding: chunked

サーバ実装はこんな感じです。 ([参考](https://echo.labstack.com/cookbook/streaming-response/))
`return c.String(200, "foo")` などとせず、 `io.WriteString` でレスポンスを書き込むのがポイントです。

```go
func transferEncodingHandler(c echo.Context) error {
  c.Response().WriteHeader(http.StatusOK)

  for _, text := range []string{"foo", "bar", "baz"} {
    _, _ = io.WriteString(c.Response(), text+"\n")
    c.Response().Flush()

    time.Sleep(200 * time.Millisecond)
  }

  return nil
}
```

クライアント実装はこんな感じです。 ([参考](https://developer.mozilla.org/ja/docs/Web/API/Streams_API/Using_readable_streams))

```js
fetch("http://localhost:1323/transfer-encoding")
  .then((response) => {
    const reader = response.body.getReader();
    return reader.read().then(function processText(result) {
      if (result.done) {
        return;
      }

      const text = new TextDecoder().decode(result.value);
      console.log(`transfer-encoding: ${text}`);

      return reader.read().then(processText);
    });
  })
  .catch((error) => console.error(error));
```

`curl` を使うと以下のようにこのエンドポイントを叩けます。
`Transfer-Encoding: chunked` というレスポンスヘッダーが確認できます。

```sh
$ curl -iN http://localhost:1323/transfer-encoding
HTTP/1.1 200 OK
Date: Tue, 09 May 2023 06:55:33 GMT
Content-Type: text/plain; charset=utf-8
Transfer-Encoding: chunked

foo
bar
baz
```

### SSE

サーバ実装はこんな感じです。
基本的には `Transfer-Encoding: chunked` と同じですが、以下2点が異なります。

- レスポンスヘッダーに `Content-Type: text/event-stream` を設定
- `event: `、 イベントの識別子、 `\n`、 `data: `、 データ、 `\n\n` の形式でレスポンスボディを書き込む

```go
func sseHandler(c echo.Context) error {
  c.Response().Header().Set("Content-Type", "text/event-stream")
  c.Response().WriteHeader(http.StatusOK)

  for _, text := range []string{"foo", "bar", "baz"} {
    _, _ = io.WriteString(c.Response(), "event: test\n")
    _, _ = io.WriteString(c.Response(), fmt.Sprintf("data: %s\n\n", text))
    c.Response().Flush()

    time.Sleep(200 * time.Millisecond)
  }

  return nil
}
```

クライアント実装はこんな感じです。 ([参考](https://developer.mozilla.org/ja/docs/Web/API/Server-sent_events/Using_server-sent_events))

サーバで `event: test` というイベントを送っているため、クライアントでは `addEventListener("test", () => {})` の形式でそのデータを受け取れます。

```js
const eventSource = new EventSource("http://localhost:1323/sse");
eventSource.addEventListener("test", (event) => {
  const text = event.data;
  console.log("sse test message:", event);
});
eventSource.addEventListener("open", (event) => {
  console.log("sse open:", event);
});
eventSource.addEventListener("close", (event) => {
  console.log("sse close:", event);
});
eventSource.addEventListener("error", (error) => {
  console.log("sse error:", error);
  eventSource.close();
});
```

`curl` を使うと以下のようにこのエンドポイントを叩けます。
`Transfer-Encoding: chunked` と `Content-Type: text/event-stream` の2つのヘッダーが確認できます。

```sh
$ curl -iN http://localhost:1323/sse
HTTP/1.1 200 OK
Content-Type: text/event-stream
Date: Tue, 09 May 2023 06:55:41 GMT
Transfer-Encoding: chunked

event: test
data: foo

event: test
data: bar

event: test
data: baz
```

### WebSocket

サーバ実装はこんな感じです。 ([参考](https://echo.labstack.com/cookbook/websocket/))
<https://pkg.go.dev/golang.org/x/net/websocket> を使用して、 `websocket.Message.Send` でちょっとずつレスポンスを書いていきます。

```go

func websocketHandler(c echo.Context) error {
  websocket.Handler(func(ws *websocket.Conn) {
    defer ws.Close()

    for _, text := range []string{"foo", "bar", "baz"} {
      _ = websocket.Message.Send(ws, text+"\n")

      time.Sleep(200 * time.Millisecond)
    }
  }).ServeHTTP(c.Response(), c.Request())

  return nil
}
```

クライアント実装はこんな感じです。([参考](https://developer.mozilla.org/ja/docs/Web/API/WebSockets_API/Writing_WebSocket_client_applications))

```js
const webSocket = new WebSocket("ws://localhost:1323/websocket");
webSocket.addEventListener("message", (event) => {
  console.log("websocket message:", event);
});
webSocket.addEventListener("open", (event) => {
  console.log("websocket open:", event);
});
webSocket.addEventListener("close", (event) => {
  console.log("websocket close:", event);
});
webSocket.addEventListener("error", (error) => {
  console.log("websocket error:", event);
});
```

`curl` を使うと以下のようにこのエンドポイントを叩けます。 ([参考](https://gist.github.com/htp/fbce19069187ec1cc486b594104f01d0))

```sh
$ curl -iN \
    --header "Connection: Upgrade" \
    --header "Upgrade: websocket" \
    --header "Host: localhost:1323" \
    --header "Origin: http://localhost:1323" \
    --header "Sec-WebSocket-Key: test" \
    --header "Sec-WebSocket-Version: 13" \
    http://localhost:1323/websocket

HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: XXXX

foo
bar
baz
```

## Lambda + Lambda Function URL

お待たせしました。やっと本題です。

レスポンスストリーミングは主に `Transfer-Encoding: chunked` で実現されています。

[Amazon Web Services ブログ](https://aws.amazon.com/jp/blogs/news/introducing-aws-lambda-response-streaming/) のサンプルコードをベースにして、ストリーミングされてることが分かりやすくなるよう、以下のようなコードを用意します。

```js
function sleep() {
  return new Promise((resolve, reject) => {
    setTimeout(resolve, 1000);
  });
}

export const handler = awslambda.streamifyResponse(async (event, responseStream, context) => {
  responseStream.setContentType("text/plain");

  for (let i = 0; i < 3; i++) {
    responseStream.write("test\n");
    await sleep();
  }

  responseStream.end();
});
```

これを Lambda にデプロイして、 invoke mode を `RESPONSE_STREAM` に設定して Lambda Function URL を有効化します。
そうして作成されたエンドポイントを `curl` で叩くと以下のようになります。
ここで `Transfer-Encoding: chunked` のレスポンスヘッダーが確認できます。

```sh
curl -iN https://xxxx.lambda-url.ap-northeast-1.on.aws/
HTTP/1.1 200 OK
Date: Tue, 09 May 2023 08:57:31 GMT
Content-Type: text/plain
Transfer-Encoding: chunked
Connection: keep-alive
x-amzn-RequestId: XXXX
X-Amzn-Trace-Id: XXXX

test
test
test
```

上記 JS コードでは `responseStream.setContentType` で Content-Type を `text/plain` に設定していました。
この Content-Type を `text/event-stream` に変更した上で、 SSE の仕様に沿ったレスポンスボディを `write` すると、このエンドポイントを SSE として扱えます。

```js
function sleep() {
  return new Promise((resolve, reject) => {
    setTimeout(resolve, 1000);
  });
}

export const handler = awslambda.streamifyResponse(async (event, responseStream, context) => {
  responseStream.setContentType("text/event-stream");

  for (let i = 0; i < 3; i++) {
    responseStream.write("event: test\ndata: test\n\n");
    await sleep();
  }

  responseStream.end();
});
```

```sh
$ curl -iN https://xxxx.lambda-url.ap-northeast-1.on.aws/
HTTP/1.1 200 OK
Date: Tue, 09 May 2023 08:56:23 GMT
Content-Type: text/plain
Transfer-Encoding: chunked
Connection: keep-alive
x-amzn-RequestId: XXXX
X-Amzn-Trace-Id: XXXX

event: test
data: test

event: test
data: test

event: test
data: test
```

## Lambda + Lambda Function URL (カスタムランタイム)

[Amazon Web Services ブログ](https://aws.amazon.com/jp/blogs/news/introducing-aws-lambda-response-streaming/) には以下の記述があります。

> レスポンスストリーミングは現在、Node.js 14.x 以降のマネージドランタイムをサポートしています。また、カスタムランタイムを使用してレスポンスストリーミングを実装することも可能です。

先程のサンプルコードでは Node.js のランタイムを使いましたが、その他 Python や Ruby では基本的には使えません。
Node.js 以外の言語を使って Lambda でレスポンスをストリーミングするには、カスタムランタイムを使うしかないです。

カスタムランタイムでレスポンスストリーミングを実装する方法は以下にドキュメント化されています。
[Custom Lambda runtimes - AWS Lambda](https://docs.aws.amazon.com/lambda/latest/dg/runtimes-custom.html#runtimes-custom-response-streaming)
(ちなみに、この情報は 2023/05/09 時点ではまだ日本語ドキュメントには記載されていません)

カスタムランタイムではクライアントにレスポンスを送信する際は、環境変数の `$AWS_LAMBDA_RUNTIME_API` で指定されたエンドポイントにレスポンスの内容を POST リクエストを送信するすることになります。
要はこの POST リクエストに以下2つのヘッダーを付けると、レスポンスストリーミングが有効になります。

- `Lambda-Runtime-Function-Response-Mode: streaming`
- `Transfer-Encoding: chunked`

今の所、これを示すサンプルとなるコードはあまり世に出ていないようですが、 [awslabs](https://github.com/awslabs/) の GitHub Organization でメンテされている Rust のカスタムランタイムを見ると実例をコードで確認できます。

- [カスタムランタイム自体の実装](https://github.com/awslabs/aws-lambda-rust-runtime/blob/fbf212f4eef8c0fd8bd87f87998239fa17bc2b23/lambda-runtime/src/streaming.rs#L221-L222)
- [その対応プルリク](https://github.com/awslabs/aws-lambda-rust-runtime/pull/628/files)
- [このカスタムランタイムを使ってレスポンスをストリーミングするサンプルコード](https://github.com/awslabs/aws-lambda-rust-runtime/blob/c08720a5728ac50abb8f8a752ca9d5d7510225f9/examples/basic-streaming-response/src/main.rs)

同様の実装を Python, Go, Ruby などで行うことによって、 Node.js 以外のランタイムでもレスポンスをストリーミングすることができるようになるはずです。
(個人的にはそんなことしなくても Node.js と同様にカスタムランタイムを使わなくて済む公式対応がそのうち実施されるだろうと予想しています)

## Lambda + API Gateway

まず、 API Gateway には HTTP API, REST API, WebSocket API の3種類があります。

[Amazon Web Services ブログ](https://aws.amazon.com/jp/blogs/news/introducing-aws-lambda-response-streaming/) の以下の記述を読む限り、このうち HTTP API と REST API ではストリーミングはできないと思われます。

> Amazon API Gateway と Application Load Balancer を使用してレスポンスペイロードをストリーミングすることはできませんが、API Gateway ではより大きなペイロードを返す機能を使用することができます。

そもそも、 HTTP API/REST API の API Gateway と Lambda を組み合わせる場合、 レスポンス全体を表すオブジェクトを `return` するようなコードを書く必要があります。

例えば、JavaScript (TypeScript) では `Record<string, unknown>`、 Python では `dict`、 Go では `map` か `struct`、Ruby では `Hash` などを `return` することになります。
例えば Go で `io.Writer` を受け取って、そこにちょっとずつレスポンスボディを書いていくというようなコードを書くことができないはずです。
なので API Gateway うんぬんの前にコードレベルでストリーミングするような表現ができないはずです。

ただし、 HTTP API と REST API でストリーミングが出来なくとも、 WebSocket API ではストリーミングができるはずです。

API Gateway (WebSocket API) + Lambda の構成で WebSocket の通信を行うやり方は新しいものではなく、関連する情報はググれば沢山見つかるのでここでは割愛します。
個人的には <https://github.com/aws-samples/simple-websockets-chat-app> のサンプルコードを見るのが最も分かりやすいと感じました。

## AppRunner

AppRunner の場合、特に工夫せずに `Transfer-Encoding: chunked` のレスポンスを返せます。
ローカルで作ったサーバがそのまま動作するはずです。

しかし、AppRunner のロードマップを管理するリポジトリの Issue を確認すると、 SSE と WebSocket は未サポートのようです。
それぞれの対応を求める Issue がありますが、どちらも 2023/05/09 時点では Close されておらず Open のままです。

- [Support for Server-SentEvents (SSE) · Issue #23 · aws/apprunner-roadmap](https://github.com/aws/apprunner-roadmap/issues/23)
- [Support web sockets · Issue #13 · aws/apprunner-roadmap](https://github.com/aws/apprunner-roadmap/issues/13)

しかし、実際に AppRunner 上に SSE のエンドポイントをデプロイしてみたところ、 SSE として解釈できるレスポンスが返ってきました。

```sh
$ curl -iN https://XXXX.ap-northeast-1.awsapprunner.com/sse
HTTP/1.1 200 OK
content-type: text/event-stream
date: Tue, 09 May 2023 09:32:51 GMT
x-envoy-upstream-service-time: 4
server: envoy
transfer-encoding: chunked

event: test
data: foo

event: test
data: bar

event: test
data: baz
```

一方、 WebSocket は 403 Forbidden が返ってきます。

```sh
$ curl -iN \
    --header "Connection: Upgrade" \
    --header "Upgrade: websocket" \
    --header "Host: xxxx.ap-northeast-1.awsapprunner.com" \
    --header "Origin: https://xxxx.ap-northeast-1.awsapprunner.com" \
    --header "Sec-WebSocket-Key: test" \
    --header "Sec-WebSocket-Version: 13" \
    https://xxxx.ap-northeast-1.awsapprunner.com/websocket
HTTP/1.1 403 Forbidden
date: Tue, 09 May 2023 09:33:46 GMT
server: envoy
connection: close
content-length: 0
```

AppRunner では Transfer-Encoding: chunked は間違いなく使えると判断して良さそうです。
SSE に関しては Issue が放置されている現状を見ると、ある日突然使えなくなってもおかしくないように思えます。
WebSocket に関しては今は使えないので、これが必要な方は [Issue](https://github.com/aws/apprunner-roadmap/issues/13) をウォッチしておくと良いのではないかと思います。

## AppSync

AppSync を使って GraphQL Subscription を使った場合、 WebSocket でストリーミングできます。

しかし、GraphQL の mutation に渡されたデータが流れてくるだけです。
それで十分な場合は問題ないですが、これまで見てきた手法と違って任意のデータを流せるわけではないです。

- [Creating generic pub/sub APIs powered by serverless WebSockets - AWS AppSync](https://docs.aws.amazon.com/appsync/latest/devguide/aws-appsync-real-time-create-generic-api-serverless-websocket.html)

## まとめ

- Lambda + API Gateway (HTTP, REST) ではストリーミングできない
- Lambda + API Gateway (WebSocket) では WebSocket でストリーミングできる
- Lambda + Lambda Function URL では Transfer-Encoding: chunked, SSE が使える
- AppRunner では Transfer-Encoding: chunked, SSE が使える
- AppSync では WebSocket が使えるケースもある
