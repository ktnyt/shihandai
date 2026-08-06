import { render } from "preact";
import { App } from "./app";
import { decodeShare } from "./lib/share";
import { loadSettings, saveSettings } from "./lib/settings";
import * as store from "./lib/store";
import "./app.css";

async function boot() {
  // 共有リンク (#p=...) が付いていたら取り込む。
  // アドレスバーには常に自分の最新状態が載るので、自分のURLの
  // リロードでは確認を出さない (ローカルと同一なら黙って続行)。
  const m = location.hash.match(/^#p=([A-Za-z0-9_-]+)$/);
  if (m) {
    const decoded = await decodeShare(m[1]);
    if (!decoded) {
      alert("共有リンクを読み込めませんでした (壊れているか、古い形式です)");
      history.replaceState(null, "", location.pathname + location.search);
    } else {
      const sameState =
        JSON.stringify(decoded.state) === JSON.stringify(store.load());
      const sameSettings =
        JSON.stringify(decoded.settings) === JSON.stringify(loadSettings());
      if (sameState && sameSettings) {
        // 自分の最新URL。何もしない
      } else if (
        confirm(
          "共有リンクの設定と進捗を読み込みますか？\nいまの設定と進捗は上書きされます。",
        )
      ) {
        saveSettings(decoded.settings);
        store.save(decoded.state);
      } else {
        // 拒否したら手元の進捗で続行し、古いURLは消す
        history.replaceState(null, "", location.pathname + location.search);
      }
    }
  }
  render(<App />, document.getElementById("app")!);
}

void boot();
