---
title: "LaunchDarkly による Feature Flags 制御の基礎"
emoji: "⛳"
type: "tech" # tech: 技術記事 / idea: アイデア
topics: ["launchdarkly"]
published: true
---

この記事では LaunchDarkly を使った Feature Flags の制御について紹介します。

## Feature Flags とは

そもそも Feature Flags がどういうものであるか、どんな種類のものがあるかについては以下の記事が参考になります。

- [Feature Toggleについて整理してみました](https://cabi99.hatenablog.com/entry/2019/10/21/144441)
- [フィーチャーフラグにはタイプ（リリース・実験・運用・許可）がある！](https://kakakakakku.hatenablog.com/entry/2022/02/01/102104)
- [Feature Toggle Types | Unleash](https://docs.getunleash.io/advanced/feature_toggle_types#feature-toggle-types)

## LaunchDarkly の用語

LaunchDarkly 上で作った Project には 1 つ以上の Environment があります。
Environment は例えば Production とか Staging とか Test とかそういうものです。

各 Environment には 0 個以上の Feature Flag があります。
ある Environment 上で Feature Flag を作ると、他の Environment 上にも同じキーの Feature Flag が作られます。
逆にある Environment 上で Feature Flag でキーをアーカイブ or 削除すると、他の Environment 上からもアーカイブ or 削除されます。

Feature Flag には以下のようなプロパティがあります。それぞれについては後ほど深堀りします。

- Individual targeting
- Rule
- Targeting するかどうか
- Permanent かどうか
- Mobile key, Client-side ID での取得を許可するか

また LaunchDarkly には User を区別する仕組みがあります。
User には `key`, `name`, `ip`, `email` などのプロパティがあります。

SDK でフラグを取得するときに User を指定することで、特定の条件にマッチしたユーザのみに別の値を返すことができます。例えば ID が 1 のユーザには 1000 を返して、メアドが `@example.com` で終わるユーザには 2000 を返して、それ以外のユーザには 0 を返すということができます。

ユーザには User と AnonymousUser があります。
AnonymousUser だとしても `key` などを用いた条件分岐は可能です。

User ごとにどんな値が返されているかブラウザ上で確認することができます。
この機能は AnonymousUser には使えません。

### Individual targeting

User の `key` を指定して、そのユーザにだけ特定の値を返すことができます。

例えば個別に問い合わせをしてきたユーザに対してのみ高度な機能を提供するなど、そのような使い方ができます。

[参考](https://docs.launchdarkly.com/home/flags/individual-targeting)

### Rule

User の `ip`, `name` などのプロパティに対して条件を指定して、その条件にマッチしたユーザにだけ特別な値を返すことができます。

例えば会社の VPN を経由した `ip` からのアクセス、または `email` が `@example.com` で終わるユーザに対してのみ特別な値を返すような設定ができます。

ここの設定でパーセンテージを指定して確率による値の変更ができます。
これによって A/B テスト、カナリアリリース、 Experiment Toggles などを実現することができます。

[参考](https://docs.launchdarkly.com/home/flags/targeting-rules)

### Targeting するかどうか

前述の Individual targeting, Rule はまとめて Targeting と呼ばれています。
これがオフのときは、全てのユーザに共通の値が返されます。

### Permanent かどうか

一般に Feature Flags には以下の種類があるとされています ([参考](https://docs.getunleash.io/advanced/feature_toggle_types#feature-toggle-types))

- Release Toggles
- Experiment Toggles
- Operational Toggles
- Kill switch
- Permission Toggles

このうち、 上の3つ (Release Toggles, Experiment Toggles, Operational Toggles) は一時的なものであり、システムを単純化するために可能な限り早く削除するのが望ましいです。
一方、 Kill switch, Permission Toggles はずっと使われるものです。

LaunchDarkly では Feature Flags が permanent (永続的) なものであるかどうか設定でき、一覧画面で permanent でないものを絞って表示することができます。
これによって permanent でないもの (いつか削除するべきもの) だけを一覧で確認することができます。

[参考](https://docs.launchdarkly.com/home/flags/settings#permanent-flags)

### Mobile key, Client-side ID での取得を許可するか

LaunchDarkly の SDK には Client SDK (Mobile SDK) と Server SDK があります。

Server SDK では SDK Key を使って API クライアントの認証を行います。
そして全てのフラグの値、ルールを取得できるようになっています。 ([参考](https://docs.launchdarkly.com/sdk/concepts/client-side-server-side#sdk-key))

Client SDK では SDK Key ではなく、Client-side ID を使って認証を行います。
Server SDK では全てのフラグの値とルールを取得できる一方、Client SDK では一部のフラグの値だけを取得することができます。
Individual targeting, Rule の設定内容にはエンドユーザのメアドなど個人情報となりうるものが含まれている可能性があるため、クライアント側でそれを取得することはできないようになっています。([参考](https://docs.launchdarkly.com/sdk/concepts/client-side-server-side#security))

一部のフラグというのが何かというと、各 Feature Flag の設定画面で `SDKs using Client-side ID`, `SDKs using Mobile key` のチェックがついているものです。
逆に言うとクライアントに存在を知られたくないフラグについてはこのチェックを外す必要があります。
この仕組みによってサーバ内部のみで使うフラグがある場合は、そのフラグの存在がクライアントに漏れることを防ぐことができます。([参考](https://docs.launchdarkly.com/sdk/concepts/client-side-server-side#client-side-id))

補足すると、ここでは分かりやすく Server SDK であるか Client SDK であるかという軸で説明しましたが、厳密には SDK Key による認証であるか、 Client-side ID による認証であるかという区別となっているはずです。
そうでなければブラウザの Developer Tools からキーを抜き出して Server SDK にそのキーを設定して、隠しているはずのフラグの存在を盗み出す…ということができてしまうはずです。

## LaunchDarkly の内部

LaunchDarkly は SSE と Fastly の高速なキャッシュパージの仕組みによってフラグの更新をリアルタイムにクライアントに反映させることができるようです。

SDK は単なる REST API 呼び出しのラッパーではありません。
まず初期化時に全てのフラグを取得して、その後は SSE ([Server Sent Event](https://developer.mozilla.org/ja/docs/Web/API/Server-sent_events/Using_server-sent_events)) でフラグの更新を検知するようになっています。 ([参考](https://docs.launchdarkly.com/sdk/concepts/getting-started#plan-for-a-large-initial-payload-from-the-streaming-endpoint))
なので管理画面でフラグを更新したときに、ユーザがページをリロードしたりアプリを再起動したりしなくともすぐにフラグの値が更新されます。
自分が簡単に確認してみた限り、1秒以内には更新されています。

最初に全てのフラグを取ってくるという仕組み、ユーザごとに Targeting できるという仕組みが合わさることによって、使用方法によっては初期化にある程度の時間がかかることが予想されます。
なので API クライアントには初期化時にタイムアウト値を設定することができます。
タイムアウトした場合はエラーにはならず、バックグラウンドで初期化処理が継続されます。
バックグラウンドで初期化処理が続いている最中でも API クライアントのオブジェクトは使用可能で、 その状態でフラグの値を取得するとコード上で指定したデフォルト値が使われます。

初期化が完了したかどうかは別途メソッド呼び出しで確認できます。例えば Go では `Initialized()` という名前、 Ruby では `initialized?` というという名前のメソッドがあります。

Server SDK ではフラグの値を取得するたびに API を叩くのではなく、初期化時に取得したルールを元に SDK 内で値を決定するため、2個のフラグを取得するとして都度通信をするわけではないです。
Client SDK でも初期化時にまとめてフラグを取得しているようで、 LaunchDarkly に送るユーザ情報が変わらない限り (Individual Targeting か Rule によってフラグの値が変わらない限り) はフラグ取得による通信は発生しないです。
このように初期化時以外に通信でブロックされるようなことはないので、 Feature Flags によるレイテンシへの悪影響は少ないはずです。

参考情報:

- <https://launchdarkly.com/how-it-works/>
- <https://www.fastly.com/jp/customers/launchdarkly>

## デフォルト値、初期値

ここまで書いてきたことのまとめですが、デフォルト値や初期値と呼べるようなものは LaunchDarkly 上には 3 種類存在します。
ややこしいので、どんな種類のものがあるかまとめます。

- SDK が初期化されていないときに使われる値 (コード上で制御する)
- Targeting がオフのときに使われる値 (ブラウザ上で制御する)
- Targeting がオンであるが、どのルールにもマッチしなかったときに使われる値 (ブラウザ上で制御する)

## 関連記事

- [LaunchDarkly による Feature Management](https://blog.studysapuri.jp/entry/2020/01/14/080000)
- [Feature Flag のメリットとプロダクトへの導入(React + TypeScript)](https://zenn.dev/kyoncy/articles/18da09f64dbc0d)
- [LaunchDarklyを入れてフィーチャーフラグの運用を改善した話](https://techblog.gaudiy.com/entry/2021/12/08/135920)
- [こんなフィーチャーフラグは気をつけろ！](https://blog.torut.tokyo/entry/2022/05/03/172348)
