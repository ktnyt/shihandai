// Package drill は練習セッションの進行と昇格・降格の判定を行う。
//
// 出題は1単語ずつ。単語が表示された瞬間から打ち終わるまでを計測し、
// ノーミスかつしきい値時間内なら成功と数える。直近の成功率が基準を
// 超えたらレベルアップする。並行安全ではない。
package drill

import (
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
	TargetKPM      float64       // 目標打鍵速度。しきい値時間の計算に使う
	ReactionBudget time.Duration // 表示から打ち始めるまでの猶予
	WindowSize     int           // 成功率を測る直近の単語数
	PromoteRate    float64       // これを上回ったらレベルアップする成功率
	DemoteAccuracy float64       // これを下回ると降格するかなの直近正答率
	MinAttempts    int           // 降格判定に必要な直近試行数
}

// DefaultConfig は既定値を返す。
func DefaultConfig() Config {
	return Config{
		TargetKPM:      120,
		ReactionBudget: time.Second,
		WindowSize:     100,
		PromoteRate:    0.95,
		DemoteAccuracy: 0.85,
		MinAttempts:    8,
	}
}

// Drill は練習全体の状態。
type Drill struct {
	Cfg   Config
	Level int
	Stats map[string]*UnitStat

	word       []string
	pos        int
	wordErrors int
	shownAt    time.Time
	results    []bool // 直近 WindowSize 単語の成否
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
	return &Drill{Cfg: cfg, Level: level, Stats: stats}
}

// Allowed は現在のレベルで使えるかなを返す。
func (d *Drill) Allowed() []string { return curriculum.For(d.Level) }

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

// Threshold は現在の単語を成功とみなす制限時間を返す。
// 反応の猶予に、打鍵数ぶんの時間を目標打鍵速度で換算して足す。
func (d *Drill) Threshold() time.Duration {
	keys := 0
	for _, u := range d.word {
		if chord, ok := naginata.ChordFor(u); ok {
			keys += chord.Count()
		} else {
			keys++
		}
	}
	typing := time.Duration(float64(keys) / d.Cfg.TargetKPM * float64(time.Minute))
	return d.Cfg.ReactionBudget + typing
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
	for _, ok := range d.results {
		if ok {
			successes++
		}
	}
	return successes, len(d.results)
}

// WordResult は単語を打ち終えたときの判定。
type WordResult struct {
	Success   bool
	Duration  time.Duration
	Threshold time.Duration
	Errors    int
	Promoted  bool
	Demoted   bool
	// WeakUnit は降格の原因になったかな。
	WeakUnit string
}

// FinishWord は単語の結果を集計し、昇格・降格を反映する。
func (d *Drill) FinishWord(now time.Time) WordResult {
	out := WordResult{
		Duration:  d.Elapsed(now),
		Threshold: d.Threshold(),
		Errors:    d.wordErrors,
	}
	out.Success = out.Errors == 0 && out.Duration <= out.Threshold

	d.results = append(d.results, out.Success)
	if len(d.results) > d.Cfg.WindowSize {
		d.results = d.results[len(d.results)-d.Cfg.WindowSize:]
	}

	// 正答率が下がったかなが出たら1つ降格する
	if d.Level > 1 {
		for _, unit := range d.Allowed() {
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

	// 窓が埋まっていて成功率が基準を上回ったらレベルアップ
	if successes, total := d.SuccessCount(); total >= d.Cfg.WindowSize &&
		float64(successes)/float64(total) > d.Cfg.PromoteRate &&
		d.Level < curriculum.MaxLevel() {
		d.Level++
		out.Promoted = true
		d.results = nil
	}
	return out
}

// resetProgress は降格時に判定の記録を捨て、連鎖降格を防ぐ。
func (d *Drill) resetProgress() {
	d.results = nil
	for _, st := range d.Stats {
		st.Recent = nil
	}
}
