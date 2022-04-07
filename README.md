かんしくん
==========

@git_tag@ ( @git_revision@ )

指定されたフォルダーを監視して、同じ名前の `*.wav` と `*.txt` が追加された時に
[ごちゃまぜドロップス](https://github.com/oov/aviutl_gcmzdrops) の外部連携 API に投げつけるプログラムです。

動作確認は AviUtl version 1.10 / 拡張編集 version 0.92 / ごちゃまぜドロップス v0.3.13 で行っています。  
プログラムの実行には Windows 7 以降が必須です。

使い方の紹介動画が sm37471880 にあります。

https://www.nicovideo.jp/watch/sm37471880

更新履歴は CHANGELOG を参照してください。

https://github.com/oov/forcepser/blob/main/CHANGELOG.md

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

1. `setting.txt-template` のファイル名を `setting.txt` に変更します。
2. `setting.txt` をテキストエディタで開き、監視対象にしたいフォルダーや、反応させたいファイル名を設定します。
3. `forcepser.exe` を起動すると黒いウィンドウが開き待機状態になります。
4. `setting.txt` の条件に一致するファイルが作成または上書きされると、自動的に `*.exo` ファイルへ整形し、拡張編集へ投げ込みます。

なお、かんしくんは同じ名前の `*.wav` と `*.txt` の2つが作成されたときにのみ反応します。  
テキストファイルを使わないケースでも必ずテキストファイルは必要です。

https://www.nicovideo.jp/watch/sm37471880

この方法での導入手順を上記動画にて解説しています。

`asas` フォルダーについて
-------------------------

`asas` フォルダーには別途配布している [Auto Save As](https://www.nicovideo.jp/watch/sm37343311) というプログラムが同梱されています。

これはプログラムの連動起動と「名前を付けて保存」ダイアログの自動処理のために使用されます。

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

設定に間違いがないはずなのに動かないときは、ウィルス対策ソフトが
`forcepser.exe`、`asas\asas.exe` をブロックしていないか確認してください。

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

かんしくんは [MSYS2](https://www.msys2.org/) + MINGW32 上で開発しています。  
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
