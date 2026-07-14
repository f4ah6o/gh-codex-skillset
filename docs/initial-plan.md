# gh-codex-skillset 仕様書

## 1. 概要

`gh-codex-skillset` は、Codex の User scope にインストールされた Skill を、Git リポジトリ単位で有効・無効化して Codex を起動する GitHub CLI 拡張である。

Codex 本体のグローバル設定は変更せず、リポジトリ内の設定ファイルを読み取り、Codex 起動時の一時設定として無効化対象を渡す。

## 2. 背景

Codex では User scope に配置された Skill が、すべてのプロジェクトで利用候補となる。

Skill 数が多い場合、次の問題が発生する。

- Skills context budget を消費する
- Skill description が短縮される
- プロジェクトと無関係な Skill が候補に含まれる
- Skill の自動選択精度が低下する可能性がある

一方で、User scope の Skill をリポジトリ単位で無効化する標準的な仕組みは限定的である。

本ツールは、Codex CLI の起動時設定を利用して、この問題を回避する。

## 3. 目的

本ツールの目的は、次の一点に限定する。

> User scope にインストールされた Codex Skill を、Git リポジトリ単位で有効・無効化する。

## 4. 対象範囲

### 4.1 対象

- Codex の User scope Skill
- Git リポジトリ単位の Skill 有効・無効設定
- Codex CLI の起動
- GitHub CLI 拡張としての提供

### 4.2 対象外

- Skill のインストール
- Skill の更新
- Skill の削除
- Project scope Skill の管理
- System scope Skill の管理
- Codex Plugin の管理
- Claude Code など Codex 以外のエージェントへの対応
- `~/.codex/config.toml` の永続的な書き換え
- Codex のプロセス外での Skill 制御
- Skill 本体の編集
- Skill description の最適化

## 5. ツール名称

### 5.1 リポジトリ名

````text
gh-codex-skillset
````

### 5.2 実行コマンド

````bash
gh codex-skillset
````

### 5.3 名称の理由

`enabler` では有効化のみを想起させるため、プロジェクト単位の Skill 集合を管理する意味を持つ `skillset` を採用する。

## 6. 基本方針

### 6.1 グローバル設定を変更しない

本ツールは、次のファイルを変更しない。

````text
~/.codex/config.toml
````

Codex の起動時にのみ、一時的な設定を適用する。

### 6.2 リポジトリに設定を保存する

Skill の有効・無効状態は、Git リポジトリ内に保存する。

これにより、設定を Git で管理・共有できる。

### 6.3 Skill 名で管理する

設定ファイルには絶対パスではなく Skill 名を保存する。

利用者ごとにホームディレクトリが異なっても、同じ設定を共有できるようにする。

### 6.4 denylist 方式を採用する

初期実装では、無効化する Skill のみを記録する。

設定に記載されていない User scope Skill は有効として扱う。

## 7. Skill 探索

### 7.1 標準探索ディレクトリ

User scope Skill は、次の2つのディレクトリから探索する。

````text
$CODEX_HOME/skills
$HOME/.agents/skills
````

`CODEX_HOME` が未設定の場合は `$HOME/.codex` を使用する。`CODEX_HOME` が設定されている場合、デフォルトの `$HOME/.codex` は追加で探索しない。

### 7.2 Skill の判定条件

上記いずれかのルート直下に `SKILL.md` が存在する場合、そのディレクトリを Skill として扱う。

````text
<skill-root>/<skill-name>/SKILL.md
````

例:

````text
$CODEX_HOME/skills/advisor-pattern/SKILL.md
$CODEX_HOME/skills/pdfs/SKILL.md
$HOME/.agents/skills/slides/SKILL.md
````

この場合、Skill 名はそれぞれ次のとおりとする。

````text
advisor-pattern
pdfs
slides
````

### 7.3 ネストされた Skill

初期実装では、直下のディレクトリのみを探索する。

次のような多階層構造は対象外とする。

````text
<skill-root>/category/example/SKILL.md
````

同じ Skill 名でも異なる物理パスにある場合は、一覧と Codex の一時設定では別エントリとして扱う。ただし、リポジトリ設定の `disabled` は名前単位のため、有効・無効状態は共有する。

### 7.4 シンボリックリンク

Skill ディレクトリまたは `SKILL.md` がシンボリックリンクであっても、実体を参照できる場合は有効な Skill として扱う。

リンク切れの場合は、`doctor` でエラーとして報告する。

## 8. Git リポジトリ判定

### 8.1 リポジトリルート

現在のディレクトリから上位へ探索し、Git リポジトリルートを取得する。

原則として、次のコマンドと同等の結果を使用する。

````bash
git rev-parse --show-toplevel
````

### 8.2 Git リポジトリ外での実行

Git リポジトリ外では、原則としてエラーとする。

MVP では次のコマンドのみ Git リポジトリ外で実行可能とする。

- `version`
- `help`
- `list --global`

## 9. 設定ファイル

### 9.1 パス

設定ファイルは、Git リポジトリルートから見て次の位置に保存する。

````text
.codex/user-skills.toml
````

### 9.2 形式

TOML 形式とする。

### 9.3 初期形式

````toml
version = 1

disabled = [
  "pdfs",
  "slides",
  "spreadsheets",
]
````

### 9.4 フィールド

| フィールド | 型 | 必須 | 説明 |
|---|---:|---:|---|
| `version` | integer | 必須 | 設定ファイルのバージョン |
| `disabled` | string array | 必須 | 無効化する User scope Skill 名 |

### 9.5 デフォルト値

設定ファイルが存在しない場合は、次の状態として扱う。

````toml
version = 1
disabled = []
````

ただし、設定変更コマンドを実行する場合は、自動的に設定ファイルを作成する。

### 9.6 不明なフィールド

未知のフィールドが存在する場合は、原則として保持する。

設定ファイル更新時に、ツールが認識しないフィールドを削除してはならない。

MVP で保持が困難な場合は、未知のフィールドが存在する設定ファイルの更新を拒否し、明示的なエラーを返す。

### 9.7 Skill 名の大文字・小文字

Skill 名は、ファイルシステム上のディレクトリ名と完全一致させる。

大文字・小文字は区別する。

## 10. コマンド仕様

### 10.1 `init`

リポジトリに設定ファイルを作成する。

````bash
gh codex-skillset init
````

作成される内容:

````toml
version = 1
disabled = []
````

#### オプション

| オプション | 説明 |
|---|---|
| `--force` | 既存の設定ファイルを初期化する |
| `--all-disabled` | 検出した全 User scope Skill を無効状態で初期化する |

#### 動作

- `.codex` ディレクトリがなければ作成する
- 設定ファイルが存在する場合、通常は上書きしない
- `--force` 指定時のみ上書きする

### 10.2 `list`

User scope Skill と、現在のリポジトリにおける状態を表示する。

````bash
gh codex-skillset list
````

出力例:

````text
STATUS    NAME               PATH
enabled   advisor-pattern    /home/user/.agents/skills/advisor-pattern/SKILL.md
disabled  pdfs               /home/user/.agents/skills/pdfs/SKILL.md
disabled  slides             /home/user/.agents/skills/slides/SKILL.md
enabled   spreadsheets       /home/user/.agents/skills/spreadsheets/SKILL.md
````

#### オプション

| オプション | 説明 |
|---|---|
| `--enabled` | 有効な Skill のみ表示 |
| `--disabled` | 無効な Skill のみ表示 |
| `--global` | リポジトリ設定を適用せず、User scope Skill 一覧のみ表示 |
| `--json` | JSON 形式で出力 |
| `--quiet` | Skill 名のみ出力 |

#### JSON 出力例

````json
[
  {
    "name": "advisor-pattern",
    "enabled": true,
    "path": "/home/user/.agents/skills/advisor-pattern/SKILL.md"
  },
  {
    "name": "pdfs",
    "enabled": false,
    "path": "/home/user/.agents/skills/pdfs/SKILL.md"
  }
]
````

### 10.3 `enable`

指定した Skill を、現在のリポジトリで有効化する。

````bash
gh codex-skillset enable advisor-pattern
````

複数指定:

````bash
gh codex-skillset enable advisor-pattern github
````

#### 動作

- 指定した Skill 名を `disabled` から削除する
- すでに有効な場合は変更しない
- 指定 Skill が存在しない場合はエラーとする
- 複数指定時は、すべて検証してから設定を更新する
- 一つでも不正な Skill があれば設定を変更しない

#### オプション

| オプション | 説明 |
|---|---|
| `--all` | 検出したすべての User scope Skill を有効化 |
| `--allow-missing` | 存在しない Skill 名も設定から削除する |

### 10.4 `disable`

指定した Skill を、現在のリポジトリで無効化する。

````bash
gh codex-skillset disable pdfs
````

複数指定:

````bash
gh codex-skillset disable pdfs slides spreadsheets
````

#### 動作

- 指定した Skill 名を `disabled` に追加する
- すでに無効な場合は変更しない
- 指定 Skill が存在しない場合はエラーとする
- 複数指定時は、すべて検証してから設定を更新する
- 一つでも不正な Skill があれば設定を変更しない

#### オプション

| オプション | 説明 |
|---|---|
| `--all` | 検出したすべての User scope Skill を無効化 |
| `--allow-missing` | 存在しない Skill 名も設定へ追加する |

### 10.5 `run`

現在のリポジトリ設定を適用して Codex を起動する。

````bash
gh codex-skillset run
````

Codex へ引数を渡す場合:

````bash
gh codex-skillset run -- exec "Issueを確認して"
````

#### 動作

1. Git リポジトリルートを取得する
2. User scope Skill を探索する
3. `.codex/user-skills.toml` を読み取る
4. 無効化対象の Skill を検証する
5. Codex 用の一時設定を生成する
6. Codex プロセスを起動する
7. Codex の終了コードをそのまま返す

#### 起動イメージ

概念的には、次の形式で Codex を起動する。

````bash
codex \
  -c 'skills.config=[
    {path="/home/user/.agents/skills/pdfs/SKILL.md",enabled=false},
    {path="/home/user/.agents/skills/slides/SKILL.md",enabled=false}
  ]' \
  "$@"
````

実際の引数形式は、Codex CLI の設定仕様に適合する形で組み立てる。

#### プロセス実行

可能であれば、子プロセスとしてではなく `exec` 相当で Codex プロセスへ置き換える。

これにより、次を保証する。

- シグナルが Codex へ直接伝達される
- Ctrl+C が正常に動作する
- 終了コードが保持される
- TTY 操作が阻害されない

Windows では、プラットフォームに適したプロセス起動方式を使用する。

#### オプション

| オプション | 説明 |
|---|---|
| `--dry-run` | Codex を起動せず、生成される設定とコマンドを表示 |
| `--strict` | 設定に存在する未インストール Skill をエラーにする |
| `--no-strict` | 未インストール Skill を警告のみとする |
| `--codex <path>` | 使用する Codex 実行ファイルを指定 |

初期値は `--strict` とする。

### 10.6 `doctor`

設定と Skill 環境を検査する。

````bash
gh codex-skillset doctor
````

#### 検査項目

- Git リポジトリであること
- 設定ファイルの構文が正しいこと
- `version` が対応範囲内であること
- `disabled` に重複がないこと
- `disabled` に存在しない Skill 名がないこと
- Skill ディレクトリが存在すること
- `SKILL.md` が存在すること
- シンボリックリンクが切れていないこと
- Codex CLI が実行可能であること
- GitHub CLI 拡張として正しい実行環境であること
- 同名 Skill が重複検出されていないこと

#### 出力例

````text
OK    repository root: /work/example
OK    config: .codex/user-skills.toml
OK    detected user skills: 12
WARN  disabled skill is not installed: old-skill
OK    codex command: /usr/local/bin/codex

1 warning, 0 errors
````

#### 終了コード

- エラーなし: `0`
- 警告のみ: `0`
- エラーあり: `1`

### 10.7 `version`

バージョンを表示する。

````bash
gh codex-skillset version
````

出力例:

````text
gh-codex-skillset 0.1.0
````

## 11. 状態の定義

### 11.1 有効

User scope に Skill が存在し、設定ファイルの `disabled` に含まれていない状態。

### 11.2 無効

User scope に Skill が存在し、設定ファイルの `disabled` に含まれている状態。

### 11.3 未インストール

設定ファイルの `disabled` に Skill 名が存在するが、User scope に対応する Skill が存在しない状態。

### 11.4 無効化設定なし

設定ファイルが存在しない、または `disabled` が空の場合、すべての User scope Skill を有効として扱う。

## 12. 設定更新

### 12.1 原子的更新

設定ファイルは直接上書きせず、一時ファイルへ書き出してから置換する。

例:

````text
.codex/user-skills.toml.tmp
.codex/user-skills.toml
````

更新途中で異常終了しても、既存設定を破損させない。

### 12.2 並び順

`disabled` の Skill 名は、辞書順に並べて保存する。

例:

````toml
version = 1

disabled = [
  "pdfs",
  "slides",
  "spreadsheets",
]
````

### 12.3 重複

重複する Skill 名は保存しない。

### 12.4 コメント

MVP では、設定ファイルを書き換える際にコメントを保持することを必須としない。

ただし、設定ファイル先頭の説明コメントはツール側で再生成してよい。

## 13. Codex 設定生成

### 13.1 基本ルール

`disabled` に含まれる Skill について、Codex 起動時に次の情報を渡す。

- `SKILL.md` の絶対パス
- `enabled = false`

### 13.2 有効 Skill

有効な Skill については、明示的な `enabled = true` を生成しない。

無効化対象のみを Codex へ渡す。

### 13.3 パスの正規化

Codex へ渡すパスは絶対パスとする。

可能であれば、次を適用する。

- `~` の展開
- `.` および `..` の解決
- シンボリックリンクの実体解決
- OS に応じたパス区切り文字の処理

### 13.4 引数の安全性

シェル文字列としてコマンドを連結せず、引数配列として Codex プロセスを起動する。

これにより、Skill 名やパスに空白・引用符・特殊文字が含まれる場合のコマンドインジェクションを防止する。

## 14. エラー処理

### 14.1 設定ファイルが壊れている

例:

````text
error: failed to parse .codex/user-skills.toml: line 4: invalid TOML
````

Codex は起動しない。

### 14.2 Skill が存在しない

例:

````text
error: user skill not found: pdf
hint: run `gh codex-skillset list`
````

### 14.3 Codex が見つからない

例:

````text
error: codex command was not found in PATH
````

### 14.4 Git リポジトリ外

例:

````text
error: current directory is not inside a Git repository
````

### 14.5 未対応バージョン

例:

````text
error: unsupported config version: 2
supported versions: 1
````

### 14.6 複数 Skill 指定時の失敗

次のような実行では、存在しない Skill が一つでもあれば設定を更新しない。

````bash
gh codex-skillset disable pdfs unknown-skill
````

出力例:

````text
error: user skill not found: unknown-skill
no changes were made
````

## 15. 終了コード

| 終了コード | 意味 |
|---:|---|
| `0` | 正常終了 |
| `1` | 一般エラー |
| `2` | コマンド引数エラー |
| `3` | 設定ファイルエラー |
| `4` | Git リポジトリエラー |
| `5` | Skill 検出・検証エラー |
| `6` | Codex 起動エラー |
| Codex の終了コード | `run` 実行後の Codex 終了結果 |

## 16. セキュリティ要件

### 16.1 グローバル設定の非変更

本ツールは、User scope の Codex 設定を永続的に変更しない。

### 16.2 任意コマンド実行の禁止

設定ファイルから外部コマンドやフックスクリプトを実行しない。

### 16.3 パス検証

Skill の `SKILL.md` は、原則として User scope Skill ルート配下に存在するもののみを対象とする。

次のようなパストラバーサルを許可しない。

````text
../../other/SKILL.md
````

### 16.4 シェル非依存

Codex 起動時は、可能な限りシェルを経由しない。

### 16.5 設定ファイルの信頼境界

リポジトリ内の `.codex/user-skills.toml` は、外部から取得したリポジトリに含まれる可能性がある。

ただし、本設定が実行できる操作は、User scope Skill を無効化することのみに限定する。

設定ファイルから次の操作を行ってはならない。

- 任意ファイルの読み込み
- 任意コマンドの実行
- 任意環境変数の設定
- Codex 以外のプログラム実行
- User scope 外の Skill 設定
- ファイル削除や変更

## 17. Git 運用

### 17.1 コミット対象

次の設定ファイルは、原則として Git 管理対象とする。

````text
.codex/user-skills.toml
````

### 17.2 チーム共有

チーム内で Skill のインストール状況が異なる可能性がある。

設定に記載された Skill が未インストールの場合、標準ではエラーとする。

ただし、運用によっては `--no-strict` を使用して警告のみにできる。

### 17.3 個人設定

個人ごとの差分を必要とする場合は、将来的にローカル上書き設定を検討する。

MVP では対応しない。

候補:

````text
.codex/user-skills.local.toml
````

このファイルを Git 管理対象外とする設計が考えられる。

## 18. 対応プラットフォーム

### 18.1 MVP

- Linux
- macOS

### 18.2 将来対応

- Windows

Windows では、次を考慮する。

- `%USERPROFILE%`
- パス区切り文字
- `codex.exe`
- プロセス置換が利用できない点
- PowerShell や cmd.exe を経由しない起動方式

## 19. 実装技術

### 19.1 言語

Go を採用する。

### 19.2 採用理由

- 単一バイナリで配布できる
- GitHub CLI 拡張と相性がよい
- クロスコンパイルが容易
- プロセス実行・シグナル処理が扱いやすい
- TOML 処理ライブラリが安定している
- 起動が高速

### 19.3 想定ライブラリ

- CLI: `cobra`
- TOML: `pelletier/go-toml`
- Git 操作: 外部 `git` コマンドまたは Go 実装
- テスト: 標準 `testing`

依存関係は最小限にする。

## 20. GitHub CLI 拡張としての構成

想定リポジトリ構成:

````text
gh-codex-skillset/
├── cmd/
│   ├── root.go
│   ├── init.go
│   ├── list.go
│   ├── enable.go
│   ├── disable.go
│   ├── run.go
│   └── doctor.go
├── internal/
│   ├── config/
│   ├── skills/
│   ├── repository/
│   ├── codex/
│   └── output/
├── main.go
├── go.mod
├── go.sum
├── README.md
├── LICENSE
└── .github/
    └── workflows/
````

## 21. 内部モデル

### 21.1 Skill

````go
type Skill struct {
    Name      string
    Directory string
    SkillFile string
    Enabled   bool
}
````

### 21.2 Config

````go
type Config struct {
    Version  int      `toml:"version"`
    Disabled []string `toml:"disabled"`
}
````

### 21.3 Codex 設定

````go
type CodexSkillConfig struct {
    Path    string
    Enabled bool
}
````

## 22. `run` 処理フロー

````text
現在ディレクトリ
    ↓
Git リポジトリルート取得
    ↓
User scope Skill 探索
    ↓
.codex/user-skills.toml 読み込み
    ↓
無効化 Skill の検証
    ↓
Skill 名を SKILL.md 絶対パスへ解決
    ↓
Codex 起動引数生成
    ↓
Codex プロセス起動
    ↓
Codex 終了コードを返却
````

## 23. 表示要件

### 23.1 通常出力

人が読める表形式を基本とする。

### 23.2 JSON 出力

自動化用途として、`list` および `doctor` は JSON 出力に対応する。

### 23.3 色

TTY の場合のみ色を使用する。

- enabled: 緑
- disabled: 黄または赤
- warning: 黄
- error: 赤

`NO_COLOR` 環境変数が設定されている場合、色を使用しない。

## 24. テスト要件

### 24.1 単体テスト

- TOML 読み込み
- TOML 書き込み
- Skill 探索
- Skill 名検証
- enabled / disabled 判定
- 設定の重複除去
- 辞書順ソート
- Codex 引数生成
- パス正規化
- 不正な設定バージョン
- 未インストール Skill
- シンボリックリンク

### 24.2 統合テスト

一時ディレクトリに次を構築して検証する。

````text
home/
└── .agents/
    └── skills/
        ├── alpha/
        │   └── SKILL.md
        └── beta/
            └── SKILL.md

repo/
├── .git/
└── .codex/
    └── user-skills.toml
````

確認項目:

- `list` の状態表示
- `enable` による設定変更
- `disable` による設定変更
- `run --dry-run` の出力
- 存在しない Skill のエラー
- 壊れた TOML のエラー
- Codex 終了コードの伝播

### 24.3 E2E テスト

スタブ Codex 実行ファイルを用意し、渡された引数を記録する。

確認項目:

- 無効化 Skill だけが設定に含まれる
- Codex 引数が保持される
- 空白を含む引数が分割されない
- シグナルと終了コードが伝播される

## 25. MVP 受入条件

以下をすべて満たした場合、MVP 完成とする。

1. User scope の Skill を一覧表示できる
2. リポジトリ単位で無効化 Skill を保存できる
3. Skill を有効化できる
4. Skill を無効化できる
5. 設定を適用して Codex を起動できる
6. `~/.codex/config.toml` を変更しない
7. Codex の終了コードを保持する
8. `--dry-run` で生成内容を確認できる
9. 存在しない Skill を検出できる
10. 設定ファイルを Git 管理できる
11. Linux および macOS で動作する
12. GitHub CLI 拡張としてインストール・実行できる

## 26. 利用例

### 26.1 初期化

````bash
cd example-project
gh codex-skillset init
````

### 26.2 Skill 一覧

````bash
gh codex-skillset list
````

### 26.3 不要な Skill を無効化

````bash
gh codex-skillset disable pdfs slides spreadsheets
````

### 26.4 Skill を再度有効化

````bash
gh codex-skillset enable spreadsheets
````

### 26.5 設定を確認

````bash
cat .codex/user-skills.toml
````

出力:

````toml
version = 1

disabled = [
  "pdfs",
  "slides",
]
````

### 26.6 Codex を起動

````bash
gh codex-skillset run
````

### 26.7 Codex サブコマンドを実行

````bash
gh codex-skillset run -- exec "このリポジトリを確認して"
````

### 26.8 起動内容のみ確認

````bash
gh codex-skillset run --dry-run
````

## 27. 将来拡張

MVP 完成後、必要に応じて次を検討する。

- allowlist 方式
- 対話型 Skill 選択 UI
- `fzf` 連携
- ローカル専用上書き設定
- 設定プロファイル
- Skill グループ
- 複数の User scope Skill ディレクトリ
- Windows 正式対応
- Codex 設定形式の自動検出
- シェルエイリアス生成
- `gh codex-skillset shell-init`
- 未インストール Skill の自動警告抑制
- Skill description やメタデータの表示
- リポジトリ言語に基づく推奨無効化候補

ただし、Skill のインストール・更新・削除は、本ツールの責務には含めない。

## 28. 非機能要件

### 28.1 性能

- Skill 数 100 件程度で、一覧・設定変更は体感待ち時間なく完了する
- Codex 起動までの追加遅延は 100 ミリ秒程度を目標とする

### 28.2 可搬性

- 設定ファイルに利用者固有の絶対パスを保存しない
- リポジトリを別環境へ clone しても設定を共有できる

### 28.3 保守性

- Codex CLI 固有の処理を `internal/codex` に隔離する
- 将来 Codex の設定形式が変わった場合も、設定ファイル形式を維持できる構造とする

### 28.4 後方互換性

設定ファイルの `version` を利用して、将来の形式変更に対応する。

バージョン 1 の設定は、可能な限り継続して読み込めるようにする。

## 29. ライセンス

オープンソースとして公開する場合は、MIT License を推奨する。

## 30. まとめ

`gh-codex-skillset` は、User scope の Codex Skill をプロジェクト単位で制御することだけに特化した GitHub CLI 拡張である。

グローバル設定を変更せず、リポジトリ内の denylist を Codex 起動時の一時設定へ変換することで、他のプロジェクトや同時実行中の Codex セッションへ影響を与えずに Skill を制御する。
