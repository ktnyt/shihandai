// Package drill は練習セッションの進行と昇格・降格の判定を行う。
package drill

import (
	"time"

	"github.com/ktnyt/shihandai/internal/curriculum"
)

// recentWindow は直近正答率を測る試行数。
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
	TargetKPM      float64 // 昇格に必要な打鍵速度 (keys per minute)
	DemoteAccuracy float64 // これを下回ると降格する直近正答率
	MinAttempts    int     // 降格判定に必要な直近試行数
}

// DefaultConfig は既定値を返す。
func DefaultConfig() Config {
	return Config{TargetKPM: 120, DemoteAccuracy: 0.85, MinAttempts: 8}
}

// Drill は練習全体の状態。
type Drill struct {
	Cfg   Config
	Level int
	Stats map[string]*UnitStat

	line       []string
	pos        int
	lineErrors int
	started    bool
	startTime  time.Time
	startKeys  int
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

// StartLine は新しい行を開始する。
func (d *Drill) StartLine(units []string) {
	d.line = units
	d.pos = 0
	d.lineErrors = 0
	d.started = false
}

// Line は現在の行を返す。
func (d *Drill) Line() []string { return d.line }

// Pos は現在の入力位置を返す。
func (d *Drill) Pos() int { return d.pos }

// LineErrors は現在の行でのミス数を返す。
func (d *Drill) LineErrors() int { return d.lineErrors }

// Expected は次に打つべきかなを返す。行末なら空文字。
func (d *Drill) Expected() string {
	if d.pos >= len(d.line) {
		return ""
	}
	return d.line[d.pos]
}

// MarkKeydown は行の最初の打鍵で計時を始める。keys は総打鍵数。
func (d *Drill) MarkKeydown(now time.Time, keys int) {
	if !d.started {
		d.started = true
		d.startTime = now
		// 最初の1打も含めて数える
		d.startKeys = keys - 1
	}
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
	// ResultLineDone は行の最後の単位を正しく入力した。
	ResultLineDone
)

// Input は確定した1単位を判定する。
func (d *Drill) Input(text string) Result {
	if d.pos >= len(d.line) {
		return ResultIgnored
	}
	switch text {
	case " ", "\b", "\n", "":
		return ResultIgnored
	}

	expected := d.line[d.pos]
	if text == expected {
		d.stat(expected).record(true)
		d.pos++
		if d.pos >= len(d.line) {
			return ResultLineDone
		}
		return ResultAdvance
	}
	d.stat(expected).record(false)
	d.lineErrors++
	return ResultError
}

func (d *Drill) stat(unit string) *UnitStat {
	s, ok := d.Stats[unit]
	if !ok {
		s = &UnitStat{}
		d.Stats[unit] = s
	}
	return s
}

// KPM は現在の行の打鍵速度を返す。keys は総打鍵数。
func (d *Drill) KPM(now time.Time, keys int) float64 {
	if !d.started {
		return 0
	}
	mins := now.Sub(d.startTime).Minutes()
	if mins <= 0 {
		return 0
	}
	return float64(keys-d.startKeys) / mins
}

// Outcome は行を打ち終えたときの判定。
type Outcome struct {
	KPM      float64
	Errors   int
	Promoted bool
	Demoted  bool
	// WeakUnit は降格の原因になったかな。
	WeakUnit string
}

// FinishLine は行の結果を集計し、昇格・降格を反映する。
func (d *Drill) FinishLine(now time.Time, keys int) Outcome {
	out := Outcome{KPM: d.KPM(now, keys), Errors: d.lineErrors}

	// 正答率が下がったかなが出たら1つ降格する
	if d.Level > 1 {
		for _, unit := range d.Allowed() {
			s, ok := d.Stats[unit]
			if !ok || len(s.Recent) < d.Cfg.MinAttempts {
				continue
			}
			if s.RecentAccuracy() < d.Cfg.DemoteAccuracy {
				d.Level--
				out.Demoted = true
				out.WeakUnit = unit
				// 降格の連鎖を防ぐため、直近の記録は捨てる
				for _, st := range d.Stats {
					st.Recent = nil
				}
				return out
			}
		}
	}

	if d.lineErrors == 0 && out.KPM >= d.Cfg.TargetKPM && d.Level < curriculum.MaxLevel() {
		d.Level++
		out.Promoted = true
	}
	return out
}
