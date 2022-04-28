---
title: "Tridactyl を使って Firefox を Vim のように制御/拡張する"
emoji: "🦶"
type: "tech" # tech: 技術記事 / idea: アイデア
topics: ["firefox"]
published: true
---

普段自分が愛用している Tridactyl という Firefox の拡張機能について紹介します。

Tridactyl は以下のような機能をもっています。

- Vim ライクなキーバインドで Firefox を操作できる
- Vim ライクな設定ファイルで設定を管理できる
- Vim ライクな設定ファイルでコマンドを定義できる

## Vim ライクなキーバインドで Firefox を操作できる

[Vimperator](http://vimperator.org/), [Vimium](https://chrome.google.com/webstore/detail/vimium/dbepggeogbaibhgnhhndojpepiihcmeb?hl=ja) などの拡張機能を使ったことがある方は、それと同じようなことができると思っていただいて大丈夫です。

例えば以下のようなことができます。

- `hjkl` でページをスクロールすること
- `d` でタブを閉じる
- `gg` でページの先頭にスクロール
- `b` でバッファを検索するように、タブをインクリメンタルに絞り込んで切り替える
- その他、 `:` を押してコマンドを検索する (VSCode の Command Palette みたいなやつ)

Vimperator や Vimium と同じように、 `f` を押すと現在表示しているページのリンクとなっている箇所がハイライトされ、各リンクにアルファベットが表示されます。
このアルファベットをタイプすることで、そのリンク先に遷移することができます。
つまりマウスを使わずにブラウジングできます。
これは[デモ](https://github.com/tridactyl/tridactyl/blob/master/doc/AMO_screenshots/trishowcase.gif)を見るのが分かりやすいと思います。

その他のショートカットキーは[README](https://github.com/tridactyl/tridactyl#default-normal-mode-bindings)から確認できます。

## Vim ライクな設定ファイルで設定を管理できる

Tridactyl は [Native messaging](https://developer.mozilla.org/ja/docs/Mozilla/Add-ons/WebExtensions/Native_messaging) の仕組みを使ってブラウザの外の世界、つまり Firefox が動いてるローカルマシンにアクセスすることができます。

これによって設定をテキストファイルで管理することができます。
dotfiles を管理している方は、その仕組みに Tridactyl を載せることができます。

設定ファイルには例えば以下のような設定ができます。

- CSS を用いた Tridactyl の UI のテーマ選択 (自作も可能)
- JavaScript を用いた独自コマンドの定義
- 既存コマンド、独自コマンドにショートカットキーを割り当てる
- 特定のドメインへの遷移などをトリガーにして、コマンドを実行するフックの定義

この仕組みを使うにはまず Firefox で `:` を押して Tridactyl のコマンドパレットを開き、 `nativeinstall` コマンドを実行します。
そうするとローカルマシンに Native messaging を処理するコマンドをインストールするスクリプトがクリップボードにコピーされるので、これをターミナルに貼り付けて実行します。

これが完了すると macOS の場合は `~/.config/tridactyl/tridactylrc` に設定ファイルを書けるようになります。

設定ファイルの例は以下で見つけられます。

- [公式の例](https://github.com/tridactyl/tridactyl/blob/master/.tridactylrc)
- [GitHub Wiki](https://github.com/tridactyl/tridactyl/wiki/Exemplar-.tridactylrc-files)
- [筆者が使っている設定](https://github.com/lambdasawa/dotfiles/blob/main/.config/tridactyl/tridactylrc)

Tridactyl のインストール後 `help` コマンドを実行することによって、どんなコマンドが使えるか確認できます。
[excmds.ts](https://github.com/tridactyl/tridactyl/blob/master/src/excmds.ts) のコメントにも同じことが書いてあります。

## Vim ライクな設定ファイルでコマンドを定義できる

例えば、設定ファイルに以下のような設定を書いたとしてます。

```js
// ~/.config/tridactyl/js/youtubeDL.js
const cmd = `cd ~/Downloads && /opt/homebrew/bin/youtube-dl -f mp4 ${location.href}`;
tri.native.run(cmd);
```

```txt
# ~/.config/tridactyl/tridactylrc
command youtubedl js -r js/youtubeDL.js
bind ,y youtubedl
```

この状態で YouTube で適当なページを開いて、 `,y` を押すと `youtube-dl` コマンドを使ってその動画をローカルに保存することができます。

ここではブラウザができること、例えば現在開いているページの URL の取得などができます。
それに加えて `tri.native.run` を使ってローカルマシン上のコマンドを実行することなどができます。
`tri` オブジェクトの中身は概ね https://github.com/tridactyl/tridactyl/tree/master/src/lib 以下のモジュールと紐付いています。

## 実例

いくつか実践的な使用例、設定例を紹介します。

### 現在開いているページを Twitter でシェア

```js
const text = encodeURIComponent(`${location.href} ${document.title}`);
window.open(`https://twitter.com/intent/tweet?text=${text}`, "_blank");
```

```
# ~/.config/tridactyl/tridactylrc
command opentwitterintent js -r js/opentwitterintent.js
bind ,t opentwitterintent
```

`js` コマンドに `-r` オプションで `~/.config/tridactyl/` からの相対パスを渡すと、 `tridactylrc` に直接 JS のコードを書くのではなく別ファイルでコマンドを管理できます。

この設定をして `,t` を押すことによって、現在のページのタイトルと URL がツイート本文として入力された状態で twitter.com を開けます。

これは Twitter のインテントの仕組みを利用しています。
インテントの仕様は以下で確認できます。

https://developer.twitter.com/en/docs/twitter-for-websites/tweet-button/guides/web-intent

### 選択範囲、あるいはページ全体を Google 翻訳に投げる

Google 翻訳はクエリストリングで翻訳したい文字列を指定することができます。
自分は選択したい文字列をこのクエリストリングに入れて新しいタブを開く処理を書いて、 `a` のショートカットキーに割り当てています。

```js
// ~/.config/tridactyl/js/translateSelection.js
const selected = window.getSelection().toString();
const encodedSelected = encodeURIComponent(selected.split(".").join(".\n\n\n"));
const url = `https://translate.google.co.jp/?sl=auto&tl=ja&text=${encodedSelected}`;
window.open(url, "_blank");
```

```
# ~/.config/tridactyl/tridactylrc
command translateselection js -r js/translateSelection.js
bind a translateselection
```

`translate.goog` 以下のサブドメインに翻訳したいページのドメインを入れることで、ページを丸ごと翻訳することができます。
自分は選択中のタブをそのドメインで開き直す処理を書いて、 `A` のショートカットキーに割り当てています。

```js
// ~/.config/tridactyl/js/translatePage.js
const subDomain = window.location.hostname.replace(/\./g, "-");
const domain = `${subDomain}.translate.goog`;
const path = location.pathname;
const searchParamas = new URLSearchParams(location.search);
searchParamas.set("_x_tr_sl", "auto");
searchParamas.set("_x_tr_tl", "ja");
searchParamas.set("_x_tr_hl", "ja");
const url = `https://${domain}/${path}?${searchParamas.toString()}`;

window.open(url, "_blank");
```

```
# ~/.config/tridactyl/tridactylrc
command translatepage js -r js/translatePage.js
bind A translatepage
```

### Tridactyl のショットカットキーを特定 URL で無効化する

```
autocmd DocStart ^https://mail.google.com mode ignore
```

この設定で URL が `^https://mail.google.com` の正規表現にマッチするページを開いた時に `mode ignore` というコマンドが自動で実行されます。

`mode ignore` コマンドでショートカットキーを無効化できます。

# 設定の再読み込み

設定ファイルを変更した後は `source` コマンドで再読み込みを行う必要があります。
毎回 `:source` とタイプするのは面倒なので `,r` にこれを割り当てています。

これは多少時間がかかる(自分の環境だと 20 秒弱) ので、その前後に `alert` を仕込んで再読み込みが終わったかどうか分かりやすくしています。

```
bind ,r composite js alert('tridactylrc loading...'); source; js alert('tridactylrc loaded!')
```

`js CODE_FOO` と書くと `CODE_FOO` の部分が `eval` されます。
`composite CMD1 CMD2...` と書くと `CMD1`, `CMD2` が順に実行されます。

## スポンサー

最後に、 Tridactyl の主要な開発者である Oliver Blanthorn さんは GitHub でスポンサーを募集しています。
https://github.com/sponsors/bovine3dom

700 USD を目標としていますが、まだ達成していません。
もしこの記事を読んで Tridactyl に興味を持って気に入っていただけたなら、スポンサーになることを検討していただけると嬉しいです。
