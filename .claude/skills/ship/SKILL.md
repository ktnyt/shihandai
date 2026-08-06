---
name: ship
description: shihandai の変更をテスト・校正してデプロイまで運ぶ手順。コードや文言を変更してコミット・公開するときに使う。
---

# 変更をデプロイまで運ぶ

## 1. テストとビルド

```sh
go vet ./... && go test -race ./...
cd web && npx vitest run && npm run build
```

片方しか触っていなくても両方通す（辞書や curriculum は共有されている）。

## 2. UI を触った場合は描画確認

```sh
cd web && npm run build
npx vite preview --port 4610 &
npx playwright-cli open http://localhost:4610/shihandai/
npx playwright-cli resize 1280 800
# 開始待ち → Space で練習中 → Esc×2 で設定パネル → resize 720 900 で狭い幅
# の4状態をスクリーンショットで確認する
npx playwright-cli close
```

文言の変更でもレイアウトは崩れる（フッターのボタン潰れの前例あり）。

## 3. 日本語の校正

コミットメッセージと、変更したドキュメント・UI 文言を
japanese-proofreader エージェントに渡し、修正を反映する。
対話の返信以外の日本語はすべて対象。

## 4. コミットとデプロイ

```sh
git add -A && git commit -m "<type>: <校正済みの説明>"
git push
gh run watch $(gh run list --limit 1 --json databaseId --jq '.[0].databaseId') --exit-status
curl -s -o /dev/null -w "%{http_code}\n" https://ktnyt.github.io/shihandai/
```

CI では Go テスト・Web テスト・GitHub Pages デプロイが走る。
サイトが 200 を返すことまで確認して完了。
