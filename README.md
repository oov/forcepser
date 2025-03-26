かんしくん
==========

@git_tag@ ( @git_revision@ )

かんしくんは、指定されたフォルダーを監視して、同じ名前の `*.wav` と `*.txt` が追加された時に
[ごちゃまぜドロップス](https://github.com/oov/aviutl_gcmzdrops) の外部連携 API に投げつけるプログラムです。

動作確認は AviUtl version 1.10 / 拡張編集 version 0.92 / ごちゃまぜドロップス v0.4.5 で行っています。  
プログラムの実行には Windows 7 以降が必須です。

注意事項
--------

かんしくん は無保証で提供されます。  
かんしくん を使用したこと及び使用しなかったことによるいかなる損害について、開発者は責任を負いません。

これに同意できない場合、あなたは かんしくん を使用することができません。  
また、プログラムの連動起動を行う場合は asas\asas.txt の注意事項も合わせて確認してください。

ダウンロード
------------

https://github.com/oov/forcepser/releases

使い方
------

かんしくんは概ね以下のような流れで使用することになります。

1. 設定ファイル `setting.txt` を準備する
  - かんしくんを使うには、まずは設定ファイルを作成しなければいけません
  - 設定ファイルは `setting.txt` という名前で保存し、`forcepser.exe` と同じ場所に置きます
2. 連携する予定の音声合成ソフトを終了しておく
  - かんしくん経由で起動していない連携予定の音声合成ソフトがある場合は、事前に終了しておく
  - 既にかんしくん経由で起動している場合は終了しなくても構いません
3. `forcepser.exe` を起動する
  - 音声合成ソフトが起動されていなければ、起動確認ダイアログが出ます
  - 何も聞かれない場合は既に起動済みか、設定が間違っている可能性があります
  - 検知可能なエラーがある場合はログに表示されているでしょう
  - ログが表示されている黒い窓はかんしくんの本体で、閉じると終了します
4. 音声合成ソフトで音声を保存する
  - かんしくんは同じ名前の `*.wav` と `*.txt` の2つが作成されたときにのみ反応します
  - 最終的にテキストが不要な用途でも、テキストファイルは必要です
  - 設定が間違っていなければかんしくんが `*.exo` を生成して拡張編集へ投げ込みます

設定ファイルは以下の方法で作成することができます。

### かんしくん設定ファイル作成ツールを使う方法

かんしくん設定ファイル作成ツール  
https://oov.github.io/forcepser/

基本的にはこちらのツールを使うことを推奨します。

1. `かんしくん設定ファイル作成ツール` でソフトの選択や設定、キャラの追加を行い、右下のボタンから設定をダウンロードします。
2. ダウンロードした `setting.txt` を `forcepser.exe` と同じフォルダーに配置します。
3. `forcepser.exe` を起動すると黒いウィンドウが開き待機状態になります。

### かんしくん設定ファイル作成ツールを使わない方法

これは今となっては古い方法ですが、柔軟な対応が必要な場合には有用です。

1. `setting.txt-template` のファイル名を `setting.txt` に変更します。
2. `setting.txt` をテキストエディタで開き、監視対象にしたいフォルダーや、反応させたいファイル名を設定します。
3. `forcepser.exe` を起動すると黒いウィンドウが開き待機状態になります。

この方法は動画での解説があります。

https://www.nicovideo.jp/watch/sm37471880

`asas` フォルダーについて
-------------------------

`asas` フォルダーには別途配布している [Auto Save As](https://www.nicovideo.jp/watch/sm37343311) というプログラムが同梱されています。

これは「名前を付けて保存」ダイアログの自動処理のために使用されます。

起動時パラメーターについて
--------------------------

`forcepser.exe` を起動する際に、以下のような引数を受け付けます。

`forcepser.exe [-v] [-m] [-prevent-clear] [settingfile]`

- `-v`
  - ログ出力を冗長にします。（主にデバッグ用）
- `-m`
  - ログ出力を着色を無効化します。
- `-prevent-clear`
  - 設定の再読み込み時に行われるログ消去を抑制します。（主にデバッグ用）
- `settingfile`
  - 設定ファイルへのパスを渡すことで、任意のファイルを設定ファイルとして読み込めます。

FAQ
---

### Q. うまく動かない

もし何らかのエラーメッセージが出ているなら、まずは読んでみましょう。  
かんしくんのログに黄色い字があれば、それが異変である可能性が高いです。

エラーメッセージは「エラーが出た」ということだけでなく、何が原因でエラーが出たのかを伝えています。  
「何を直せばいいのか」は書いていませんが、改善のヒントにはなります。

### Q. 「設定の中に問題があるため一部の機能が無効化されています」と出る

設定の中に問題が見つかったため一部の機能が無効化されました。  
ログを遡ると黄色いテキストが出てくるはずです。

#### 「対象EXE が見つからないため設定を無視します」と出た

音声合成ソフトのインストール場所の入力が間違っているのが原因でエラーが出ています。  
あなたがご自身でどこにインストールしたのかを確認し、正しい場所を入力する必要があります。

#### 「対象フォルダー が見つからないため設定を無視します」と出た

監視フォルダーとして設定したフォルダーが実際には存在しないのが原因でエラーが出ています。  
正しい場所を設定してください。

### Q. 「ごちゃまぜドロップス v0.3 以降がインストールされた AviUtl が検出できませんでした」というエラーが出る

- ごちゃまぜドロップスが入っていないかバージョンが古い
- AviUtl を起動していない
- 外部連携APIが有効になっていない
- セキュリティソフトなど外部のプログラムにブロックされている

大抵の原因はこのどれかです。
以下の順序で確認すると良いでしょう。

#### 1. ごちゃまぜドロップス v0.4.5 以降に更新する  

AviUtl の `その他` メニューから `プラグインフィルタ情報` を開くとバージョンが確認できます。

- PSDToolKit を使っている -> PSDToolKit を更新してください
- PSDToolKit を使っていない -> ごちゃまぜドロップスを更新してください

#### 2. AviUtl を普通に起動する

AviUtl もかんしくんも `管理者として実行` を使わずに起動するようにしましょう。

- トラブルの回避策としてたまに `管理者として実行` を挙げる人がいますが、危険なのでやめましょう
- これで解決できる例は稀で、他の問題を引き起こす原因になることもあり、闇雲に行うべきではありません

#### 3. 外部連携APIの状態を確認する

外部連携APIが正しく準備できているか確認しましょう。

- AviUtl の `表示` メニューから `ごちゃまぜドロップスの表示` を選ぶと設定画面が表示されます
- `外部連携API` の枠が存在しない -> 最初のステップをやり直してください
- `外部連携API` のところが `状態: 稼働中` -> 外部連携APIが使用可能です
  - この状態なのに動かない場合は、外部のプログラムにブロックされている可能性が高そうです
- `外部連携API` のところが `状態: エラーが発生しました` -> 対応が必要です
  - `外部連携APIを使用する` のチェックを外して付け直すと解決します
  - 主に AviUtl を多重起動した場合に起こる現象です（外部連携APIは複数同時に有効化できない）

#### 4. 外部のプログラムにブロックされていないか確認しましょう

あなたが飼い犬をしっかり躾けられているか確認しましょう。

しかしここで紹介する手順は要するに `かんしくん` などを **安全なプログラム** だと認識させるための作業です。  
例えばあなたが公式の配布サイト以外からダウンロードした場合には安全でない可能性があります。  
変なサイトからダウンロードしたとか、他者が作成した自動インストールツールなどを経由した場合には特に注意が必要です。

以下の手順は自己責任で実行してください。

1. エクスプローラーから `forcepser.exe` や `asas\asas.exe` のプロパティを表示し `ブロックの解除` ができているか確認してください  
  - インターネットからダウンロードしたファイルは `Mark of the Web` という仕組みによりブロックされることがあります
  - 信頼できるファイルなのにブロックされている場合はプロパティから `ブロックの解除` を行う必要があります
2. スマートアプリコントロールを無効化する
  - 現在は配布プログラムへの署名を行っていないため、スマートアプリコントロールが有効だと正常に動作しない可能性があります
3. `forcepser.exe` や `asas\asas.exe` が Windows Defender などにブロックされていないか確認しましょう
  - 除外に登録したり、一時的に無効化して試してみるなどの対策を取ってみてください
4. それでも駄目なら
  - 3 の手順が不十分か、ここで想定できない他の問題が発生している可能性があります
  - 詳しい人に聞くか、他のアプリケーションを検討してください

### Q. 「AviUtl のプロジェクトファイルがまだ保存されていないため処理を続行できません」というエラーが出る

AviUtl のプロジェクトファイルがまだ保存されていないため処理を続行できませんでした。  
AviUtl のプロジェクトファイルを保存してください。

### Q. 音声を保存すると「一致するルールが見つかりませんでした」と出る

ここでは `かんしくん設定ファイル作成ツール` を利用していることを前提に解説しています。  
使っていない場合はご自身でルールを記述されているので、ここでの解説は当てはまりません。

かんしくんではファイル名を元に、どの振り分けルールを適用すべきなのかを判断します。  
このエラーが出るということは、ファイル名と設定済みのルールの間に齟齬があるということです。

このメッセージが出たとき、すぐ上にはファイル名がフルパスで表示されています。  
その末尾は例えば `～～\123456789_キャラ名_こんにちは.wav` のような形になっているはずです。

#### ファイル名が `123456789_キャラ名_こんにちは.wav` のような形式の場合

- 事前に設定したキャラ名とファイル名のキャラ名が正しく一致しているか確認してください
  - スペースの有無の違い
  - カッコやスペースが全角と半角で異なっていないか
  - アプリケーションのバージョンアップで名前の表記が変更された場合もあるかもしれません
  - `かんしくん設定ファイル作成ツール` で用意したプリセットが間違っているケースもあるかもしれません
- ユーザープリセットを使う場合は `キャラ名 - 喜び` のような形で命名してください(VOICEROID2, A.I.VOICE, A.I.VOICE2)
  - この形式で入力した場合は `キャラ名` の部分をキャラクター名として認識します
  - それ以外の形式で入力すると全体を名前として認識します
- キャラ名ではなく `Talk1` という名前になっている(Voisona Talk)
  - Voisona Talk ではキャラクター名ではなくトラック名が使用されます
  - `かんしくん設定ファイル作成ツール` で指定した名前とトラック名を一致させてください

#### ファイル名が `VOICEROID2_123456789.wav` みたいな感じでキャラ名がない(VOICEROID2のみ)

VOICEROID2 には出力ファイル名にキャラクター名を含める機能がありません。
そのため、セリフにボイスプリセット名を含めることで代用します。

- セリフ入力時に `Ctrl+I` を押したりキャラ名をセリフ入力欄にドラッグして、ボイスプリセット指定を含めてください
- ユーザー定義のボイスプリセットは `キャラ名 - 喜び` のような形で命名すれば認識します。
- それ以外の形式で入力したユーザー定義プリセットは、そのようなキャラクター名として認識されます。

### Q. 「読み取りに失敗しました: EOF」というエラーが出る

音声合成ソフトがファイルを書き込んでいる最中にかんしくんが動作してエラーが出ることがあります。

かんしくんには自動リトライ機構があり、失敗しても何度かやり直しするようになっています。  
ログに「2回目」などの表示があり、上手くファイルが処理されるようなら、このエラーは無視して問題ありません。

### Q. ドロップされるファイルの順番がおかしい

監視対象のフォルダーがある場所のファイルシステムが FAT や FAT32 の場合、ファイルの更新日時が2秒単位でしか保持されません。  
このようなケースでは、音声合成ソフトが次々とファイルを生成する場合に正しい順番で処理できない可能性があります。

音声ファイルを一括生成して処理を行う場合は NTFS や exFAT などの、更新日時の精度が高いファイルシステムを利用してみてください。

バイナリのビルドについて
------------------------

かんしくんは [MSYS2](https://www.msys2.org/) + CLANG64 上で開発しています。  
ビルド方法や必要になるパッケージなどは [GitHub Actions の設定ファイル](https://github.com/oov/forcepser/blob/main/.github/workflows/releaser.yml) を参考にしてください。

Credits
-------

かんしくん is made possible by the following open source softwares.

### The Go Programming Language

https://golang.org/  
https://golang.org/x/sys/windows  
https://golang.org/x/text/encoding

Copyright (c) 2009 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
&quot;AS IS&quot; AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

### CLI Color

https://github.com/gookit/color

The MIT License (MIT)

Copyright (c) 2016 inhere

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

### fsnotify

https://github.com/fsnotify/fsnotify

Copyright (c) 2012 The Go Authors. All rights reserved.  
Copyright (c) 2012 fsnotify Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

### GopherLua

https://github.com/yuin/gopher-lua

The MIT License (MIT)

Copyright (c) 2015 Yusuke Inuzuka

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

### gluare

https://github.com/yuin/gluare

MIT License

Copyright (c) 2017 Yusuke Inuzuka

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

### go-toml

https://github.com/pelletier/go-toml

The MIT License (MIT)

Copyright (c) 2013 - 2017 Thomas Pelletier, Eric Anderton

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

### audio

https://github.com/oov/audio

The MIT License (MIT)

Copyright (c) 2013 Masanobu YOSHIOKA

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
