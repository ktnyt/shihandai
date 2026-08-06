// Package drill は練習セッションの進行と昇格・降格の判定を行う。
//
// 出題は1単語ずつ。単語が表示された瞬間から打ち終わるまでを計測する。
// 昇格は直近の窓の打鍵速度 (kpm) とミス率の両方が基準を満たしたとき。
// 並行安全ではない。
package drill

import (
	"slices"
	"time"

	"github.com/ktnyt/shihandai/internal/curriculum"
	"github.com/ktnyt/shihandai/internal/naginata"
)

// recentWindow は1つのかなの直近正答率を測る試行数。
const recentWindow = 12

// UnitStat は1つのかなの成績。
type UnitStat struct {
	Attempts int    `json:"attempts"`
	Errors   int    `json:"errors"`
	Recent   []bool `json:"recent"` // 直近の試行の成否
}

func (s *UnitStat) record(ok bool) {
	s.Attempts++
	if !ok {
		s.Errors++
	}
	s.Recent = append(s.Recent, ok)
	if len(s.Recent) > recentWindow {
		s.Recent = s.Recent[len(s.Recent)-recentWindow:]
	}
}

// RecentAccuracy は直近の正答率を返す。試行がなければ1を返す。
func (s *UnitStat) RecentAccuracy() float64 {
	if len(s.Recent) == 0 {
		return 1
	}
	ok := 0
	for _, r := range s.Recent {
		if r {
			ok++
		}
	}
	return float64(ok) / float64(len(s.Recent))
}

// Config は判定の調整項目。
type Config struct {
	TargetKPM       float64       // 昇格に必要な打鍵速度 (keys per minute)
	MaxMissRate     float64       // 昇格できるミス率の上限
	ReactionBudget  time.Duration // 表示から打ち始めるまでの猶予。kpm の計算で引く
	WindowSize      int           // 判定に使う直近の単語数
	MinNewKanaWords int           // 昇格までに打つ、新出かなを含む語の最低数
	DemoteAccuracy  float64       // これを下回ると降格するかなの直近正答率
	MinAttempts     int           // 降格判定に必要な直近試行数
}

// DefaultConfig は既定値を返す。
// 降格はゆるめにしてある。新しいかなが入ると既存のかなも文脈が変わって
// 一時的に難しくなるので、直近12回が埋まった上で4ミス以上（70%未満）で
// はじめて降格する。
func DefaultConfig() Config {
	return Config{
		TargetKPM:       120,
		MaxMissRate:     0.05,
		ReactionBudget:  500 * time.Millisecond,
		WindowSize:      100,
		MinNewKanaWords: 50,
		DemoteAccuracy:  0.70,
		MinAttempts:     recentWindow,
	}
}

// WordRecord は判定の窓に残る、1単語分の記録。
type WordRecord struct {
	Success bool          `json:"success"`
	Units   int           `json:"units"`  // 正しく打ったかなの数
	Keys    int           `json:"keys"`   // 打鍵数 (同時押しは複数と数える)
	Errors  int           `json:"errors"` // ミス入力の数
	Typing  time.Duration `json:"typing"` // 反応の猶予を引いた打鍵時間
}

// Drill は練習全体の状態。
type Drill struct {
	Cfg   Config
	Level int
	Stats map[string]*UnitStat

	word          []string
	pos           int
	wordErrors    int
	shownAt       time.Time
	records       []WordRecord // 直近 WindowSize 単語の記録
	newKanaWords  int          // このレベルで打った、新出かなを含む語の数
	newKanaSupply int          // 新出かなを含む語が辞書に何語あるか。負なら不明
}

// New は練習状態を作る。
func New(cfg Config, level int, stats map[string]*UnitStat) *Drill {
	if stats == nil {
		stats = map[string]*UnitStat{}
	}
	if level < 1 {
		level = 1
	}
	if level > curriculum.MaxLevel() {
		level = curriculum.MaxLevel()
	}
	return &Drill{Cfg: cfg, Level: level, Stats: stats, newKanaSupply: -1}
}

// Allowed は現在のレベルで使えるかなを返す。
func (d *Drill) Allowed() []string { return curriculum.For(d.Level) }

// Stage は現在のレベルの出題条件を返す。
func (d *Drill) Stage() curriculum.Stage { return curriculum.StageFor(d.Level) }

// Newest はいちばん新しく増えたかなを返す。
func (d *Drill) Newest() string {
	allowed := d.Allowed()
	return allowed[len(allowed)-1]
}

// NewKanaWords はこのレベルで打った、新出かなを含む語の数を返す。
func (d *Drill) NewKanaWords() int { return d.newKanaWords }

// SetNewKanaSupply は新出かなを含む語が辞書に何語あるかを教える。
// 語彙の少ないかなでゲートが満たせなくなるのを防ぐのに使う。
func (d *Drill) SetNewKanaSupply(n int) { d.newKanaSupply = n }

// supplyFactor は語彙の少ないかなのゲートの倍率。
// 辞書に4語しかなければ 4×5=20 語打てば足りる。
const supplyFactor = 5

// GateTarget は昇格までに打つべき、新出かなを含む語の数を返す。
// 最初の5文字の段階では全部が新出なので、ゲートはかけない。
func (d *Drill) GateTarget() int {
	if len(d.Allowed()) == len(curriculum.For(1)) {
		return 0
	}
	target := d.Cfg.MinNewKanaWords
	if d.newKanaSupply >= 0 {
		target = min(target, d.newKanaSupply*supplyFactor)
	}
	return target
}

// Progress は保存のために判定の窓とゲートのカウンタを返す。
func (d *Drill) Progress() (records []WordRecord, newKanaWords int) {
	return d.records, d.newKanaWords
}

// SetProgress は保存されていた進捗を復元する。
func (d *Drill) SetProgress(records []WordRecord, newKanaWords int) {
	if len(records) > d.Cfg.WindowSize {
		records = records[len(records)-d.Cfg.WindowSize:]
	}
	d.records = records
	d.newKanaWords = max(newKanaWords, 0)
}

// StartWord は新しい単語を出題する。now は表示した時刻。
func (d *Drill) StartWord(units []string, now time.Time) {
	d.word = units
	d.pos = 0
	d.wordErrors = 0
	d.shownAt = now
}

// Word は現在の単語を返す。
func (d *Drill) Word() []string { return d.word }

// Pos は現在の入力位置を返す。
func (d *Drill) Pos() int { return d.pos }

// WordErrors は現在の単語でのミス数を返す。
func (d *Drill) WordErrors() int { return d.wordErrors }

// Expected は次に打つべきかなを返す。単語の終わりなら空文字。
func (d *Drill) Expected() string {
	if d.pos >= len(d.word) {
		return ""
	}
	return d.word[d.pos]
}

// Elapsed は表示からの経過時間を返す。
func (d *Drill) Elapsed(now time.Time) time.Duration {
	if d.shownAt.IsZero() {
		return 0
	}
	return now.Sub(d.shownAt)
}

// wordKeys は現在の単語の打鍵数を返す。
func (d *Drill) wordKeys() int {
	keys := 0
	for _, u := range d.word {
		if chord, ok := naginata.ChordFor(u); ok {
			keys += chord.Count()
		} else {
			keys++
		}
	}
	return keys
}

// Result は1回の入力の判定結果。
type Result int

const (
	// ResultIgnored は判定対象外の入力（空白、制御文字など）。
	ResultIgnored Result = iota
	// ResultAdvance は正しい入力で1単位進んだ。
	ResultAdvance
	// ResultError は誤った入力。位置は進まない。
	ResultError
	// ResultWordDone は単語の最後の単位を正しく入力した。
	ResultWordDone
)

// Input は確定した1単位を判定する。
func (d *Drill) Input(text string) Result {
	if d.pos >= len(d.word) {
		return ResultIgnored
	}
	switch text {
	case " ", "\b", "\n", "":
		return ResultIgnored
	}

	expected := d.word[d.pos]
	if text == expected {
		d.stat(expected).record(true)
		d.pos++
		if d.pos >= len(d.word) {
			return ResultWordDone
		}
		return ResultAdvance
	}
	d.stat(expected).record(false)
	d.wordErrors++
	return ResultError
}

func (d *Drill) stat(unit string) *UnitStat {
	// 手で編集された進捗ファイル由来の nil 値にも耐える
	s := d.Stats[unit]
	if s == nil {
		s = &UnitStat{}
		d.Stats[unit] = s
	}
	return s
}

// SuccessCount は直近の成功数と試行数を返す。
func (d *Drill) SuccessCount() (successes, total int) {
	for _, r := range d.records {
		if r.Success {
			successes++
		}
	}
	return successes, len(d.records)
}

// WindowKPM は直近の窓の打鍵速度 (keys per minute) を返す。
// 各単語の経過時間から反応の猶予を引いた分を打鍵時間とみなす。
func (d *Drill) WindowKPM() float64 {
	keys := 0
	var typing time.Duration
	for _, r := range d.records {
		keys += r.Keys
		typing += r.Typing
	}
	if typing <= 0 {
		return 0
	}
	return float64(keys) / typing.Minutes()
}

// MissRate は直近の窓のミス率を返す。
// 打ったかな (正解とミスの合計) のうちミスの割合。
func (d *Drill) MissRate() float64 {
	units, errors := 0, 0
	for _, r := range d.records {
		units += r.Units
		errors += r.Errors
	}
	if units+errors == 0 {
		return 0
	}
	return float64(errors) / float64(units+errors)
}

// WordResult は単語を打ち終えたときの判定。
type WordResult struct {
	Success  bool
	Duration time.Duration
	Errors   int
	Promoted bool
	// KanaAdded は昇格でかなが増えたことを示す。false の昇格は長さの解放。
	KanaAdded bool
	Demoted   bool
	// WeakUnit は降格の原因になったかな。
	WeakUnit string
}

// FinishWord は単語の結果を集計し、昇格・降格を反映する。
func (d *Drill) FinishWord(now time.Time) WordResult {
	out := WordResult{
		Duration: d.Elapsed(now),
		Errors:   d.wordErrors,
	}
	// 成功はミスの有無だけで決める。速さは窓の kpm で別に判定する
	out.Success = out.Errors == 0

	// 反応の猶予より速く打ち終えた語で速度が発散しないよう下限を置く
	typing := max(out.Duration-d.Cfg.ReactionBudget, 10*time.Millisecond)
	d.records = append(d.records, WordRecord{
		Success: out.Success,
		Units:   len(d.word),
		Keys:    d.wordKeys(),
		Errors:  out.Errors,
		Typing:  typing,
	})
	if len(d.records) > d.Cfg.WindowSize {
		d.records = d.records[len(d.records)-d.Cfg.WindowSize:]
	}
	if slices.Contains(d.word, d.Newest()) {
		d.newKanaWords++
	}

	// 正答率が下がったかなが出たら1つ降格する。
	// 覚えている最中のいちばん新しいかなは対象外。
	// 時間超過は打ち間違いではないので、ここには影響しない
	// （かなの統計は入力の正誤だけを記録している）。
	if d.Level > 1 {
		allowed := d.Allowed()
		for _, unit := range allowed[:len(allowed)-1] {
			s := d.Stats[unit]
			if s == nil || len(s.Recent) < d.Cfg.MinAttempts {
				continue
			}
			if s.RecentAccuracy() < d.Cfg.DemoteAccuracy {
				d.Level--
				out.Demoted = true
				out.WeakUnit = unit
				d.resetProgress()
				return out
			}
		}
	}

	// 窓が埋まっていて、打鍵速度とミス率の両方が基準を満たし、
	// 新出かなを含む語も十分に打っていたらレベルアップ
	if len(d.records) >= d.Cfg.WindowSize &&
		d.WindowKPM() >= d.Cfg.TargetKPM &&
		d.MissRate() <= d.Cfg.MaxMissRate &&
		d.newKanaWords >= d.GateTarget() &&
		d.Level < curriculum.MaxLevel() {
		before := len(d.Allowed())
		d.Level++
		out.Promoted = true
		out.KanaAdded = len(d.Allowed()) > before
		d.records = nil
		d.newKanaWords = 0
	}
	return out
}

// resetProgress は降格時に判定の記録を捨て、連鎖降格を防ぐ。
func (d *Drill) resetProgress() {
	d.records = nil
	d.newKanaWords = 0
	for _, st := range d.Stats {
		st.Recent = nil
	}
}
