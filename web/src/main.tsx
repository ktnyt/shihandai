import { render } from "preact";
import { App } from "./app";
import { decodeShare } from "./lib/share";
import { saveSettings } from "./lib/settings";
import * as store from "./lib/store";
import "./app.css";

async function boot() {
  // 共有リンク (#p=...) が付いていたら取り込みを提案する
  const m = location.hash.match(/^#p=([A-Za-z0-9_-]+)$/);
  if (m) {
    const decoded = await decodeShare(m[1]);
    history.replaceState(null, "", location.pathname + location.search);
    if (
      decoded &&
      confirm(
        "共有リンクの設定と進捗を読み込みますか？\nいまの設定と進捗は上書きされます。",
      )
    ) {
      saveSettings(decoded.settings);
      store.save(decoded.state);
    } else if (!decoded) {
      alert("共有リンクを読み込めませんでした (壊れているか、古い形式です)");
    }
  }
  render(<App />, document.getElementById("app")!);
}

void boot();
