import { defineConfig } from "vite";
import preact from "@preact/preset-vite";
import { fileURLToPath } from "node:url";

// 辞書はGo版と共有する (tools/mkdict が生成する)
const dictPath = fileURLToPath(
  new URL("../internal/dict/words.txt", import.meta.url),
);

export default defineConfig({
  base: "/shihandai/",
  plugins: [preact()],
  resolve: {
    // ?raw クエリが付いても解決できるよう正規表現で合わせる
    alias: [{ find: /^@dict\/words\.txt/, replacement: dictPath }],
  },
  server: {
    fs: {
      // リポジトリ直下の internal/dict/words.txt を読むため
      allow: [".."],
    },
  },
  test: {
    environment: "node",
  },
});
