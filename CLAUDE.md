# shihandai

薙刀式かな入力の段階練習ツール。Go 製 CLI と `web/` の SPA（Preact + Vite）の2本立て。
公開先は https://ktnyt.github.io/shihandai/ で、main へ push すると GitHub Actions で
テストとデプロイが走る。

## コマンド

```sh
go test -race ./...                 # CLI側のテスト
cd web && npx vitest run            # Web側のテスト
cd web && npm run build             # 型チェック込みのビルド
cd web && npm run dev               # 開発サーバ
go run ./cmd/shihandai              # CLI版の起動
```

## 構成

- `internal/naginata` — 薙刀式v15の配列テーブルと同時押し判定（CLI はタイミングウィンドウ方式）
- `internal/curriculum` — レベル定義（かな128段階 × 長さ4段階 = 512）
- `internal/lesson` — 辞書からの単語選択（新出40% / 苦手20% / 通常40%）
- `internal/drill` — 判定と昇格・降格
- `internal/dict` — mozc 由来の頻度順単語リスト（`tools/mkdict` で再生成）
- `web/src/lib` — 上記の TypeScript 移植。エンジンだけは keyup を使う QMK 準拠の
  リリース確定方式で、CLI 版と実装が異なる
- 辞書 `internal/dict/words.txt` は Web 側から vite のエイリアスで直接参照している

## 約束ごと

- 判定仕様（昇格・降格の条件、出題の割合、プリセットの値）はユーザーと詰めた結果なので、
  指示なく変えない。変えるときは Go と Web の両方のテストを更新する
- 対話の返信以外の日本語（コミットメッセージ、ドキュメント、UI 文言）は、
  japanese-proofreader エージェントを通してから保存・コミットする
- 文体: Web のメッセージ類は敬体、ラベルと設定の説明文は常体、
  練習画面のひらがな演出（ながさ、つぎ、あたらしいかな）は維持、CLI は常体
- JSX は行頭の全角スペースをトリムする。語間の間隔は `{"　…"}` のような
  明示的な文字列式で入れる
- UI を変更したら playwright-cli で確認する（`npm run build` →
  `npx vite preview` → 開始待ち・練習中・設定パネル・狭い幅の4状態を撮る）
- リリースの手順は `.claude/skills/ship` に従う
