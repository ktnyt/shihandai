// mkdict は mozc のOSS辞書から練習用の単語リストを作る。
//
// 使い方:
//
//	go run ./tools/mkdict -dict-dir <dictionary0*.txt と id.def のあるディレクトリ> \
//	  -out internal/dict/words.txt -n 20000
//
// 一般語の基本形だけを残し、薙刀式で打てる読みを
// コスト（小さいほど高頻度）順に出力する。
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/ktnyt/shihandai/internal/curriculum"
	"github.com/ktnyt/shihandai/internal/lesson"
)

func main() {
	var (
		dir = flag.String("dict-dir", ".", "dictionary0*.txt と id.def のあるディレクトリ")
		out = flag.String("out", "internal/dict/words.txt", "出力先")
		n   = flag.Int("n", 20000, "残す語数")
	)
	flag.Parse()

	if err := run(*dir, *out, *n); err != nil {
		fmt.Fprintln(os.Stderr, "エラー:", err)
		os.Exit(1)
	}
}

// loadIDs は id.def を読み、練習語彙にふさわしい品詞IDの集合を返す。
func loadIDs(path string) (map[int]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("id.def を開けない: %w", err)
	}
	defer f.Close()

	ok := map[int]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		id, feature, found := strings.Cut(sc.Text(), " ")
		if !found {
			continue
		}
		fields := strings.Split(feature, ",")
		if len(fields) < 6 {
			continue
		}
		if !keepPOS(fields) {
			continue
		}
		v, err := strconv.Atoi(id)
		if err != nil {
			continue
		}
		ok[v] = true
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("id.def の読み込みに失敗: %w", err)
	}
	return ok, nil
}

// keepPOS は品詞素性 [品詞1, 品詞2, 品詞3, 品詞4, 活用型, 活用形, ...] を判定する。
func keepPOS(f []string) bool {
	pos1, pos2, cform := f[0], f[1], f[5]

	// 活用する語は基本形だけ残す
	if cform != "*" && cform != "基本形" {
		return false
	}
	switch pos1 {
	case "名詞":
		switch pos2 {
		case "一般", "サ変接続", "副詞可能", "形容動詞語幹", "代名詞":
			return true
		}
		return false
	case "動詞", "形容詞":
		return pos2 == "自立"
	case "副詞", "連体詞", "接続詞", "感動詞":
		return true
	}
	return false
}

type entry struct {
	reading string
	cost    int
}

func run(dir, outPath string, n int) error {
	ids, err := loadIDs(filepath.Join(dir, "id.def"))
	if err != nil {
		return err
	}
	files, err := filepath.Glob(filepath.Join(dir, "dictionary0*.txt"))
	if err != nil || len(files) == 0 {
		return fmt.Errorf("辞書ファイルが見つからない: %s", dir)
	}

	// 練習で扱う全単位。これで分割できない読みは打てないので捨てる。
	universe := curriculum.For(curriculum.MaxLevel())

	best := map[string]int{} // 読み → 最小コスト
	total := 0
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("辞書を開けない: %w", err)
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			total++
			fields := strings.Split(sc.Text(), "\t")
			if len(fields) < 5 {
				continue
			}
			reading := fields[0]
			lid, err1 := strconv.Atoi(fields[1])
			cost, err2 := strconv.Atoi(fields[3])
			if err1 != nil || err2 != nil || !ids[lid] {
				continue
			}
			units, ok := lesson.Segment(reading, universe)
			if !ok || len(units) < 2 || len(units) > 7 {
				continue
			}
			if prev, seen := best[reading]; !seen || cost < prev {
				best[reading] = cost
			}
		}
		f.Close()
		if err := sc.Err(); err != nil {
			return fmt.Errorf("辞書の読み込みに失敗: %w", err)
		}
	}

	entries := make([]entry, 0, len(best))
	for reading, cost := range best {
		entries = append(entries, entry{reading, cost})
	}
	slices.SortFunc(entries, func(a, b entry) int {
		if a.cost != b.cost {
			return a.cost - b.cost
		}
		return strings.Compare(a.reading, b.reading)
	})
	if len(entries) > n {
		entries = entries[:n]
	}

	var b strings.Builder
	b.WriteString("# shihandai 練習用単語リスト（頻度順）\n")
	b.WriteString("# mozc の OSS 辞書 (data/dictionary_oss) から tools/mkdict で生成した。\n")
	b.WriteString("# 元データ: mozc (c) Google Inc. (BSD-3-Clause)\n")
	for _, e := range entries {
		b.WriteString(e.reading)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(outPath, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("出力の書き込みに失敗: %w", err)
	}

	fmt.Printf("%d 行から %d 語を抽出し、%d 語を書き出した\n", total, len(best), len(entries))
	return nil
}
