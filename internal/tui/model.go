// Package tui は bubbletea 製の練習画面を提供する。
package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ktnyt/shihandai/internal/drill"
	"github.com/ktnyt/shihandai/internal/lesson"
	"github.com/ktnyt/shihandai/internal/naginata"
	"github.com/ktnyt/shihandai/internal/store"
)

const tickInterval = 30 * time.Millisecond

// Model は画面全体の状態。
type Model struct {
	engine *naginata.Engine
	drill  *drill.Drill
	gen    *lesson.Generator

	statePath string

	paused    bool
	leveledUp bool // レベルアップ画面を表示中
	kanaAdded bool // 直近の昇格でかなが増えた（false なら長さの解放）
	width     int
	message   string
	flash     string
	err       error
}

// New は画面を作り、最初の単語を出題する。
func New(engine *naginata.Engine, d *drill.Drill, gen *lesson.Generator, statePath string) (Model, error) {
	m := Model{
		engine:    engine,
		drill:     d,
		gen:       gen,
		statePath: statePath,
	}
	if err := m.newWord(time.Now()); err != nil {
		return Model{}, err
	}
	return m, nil
}

type tickMsg time.Time

type saveErrMsg struct{ err error }

func (m Model) tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Init は定期処理を開始する。
func (m Model) Init() tea.Cmd { return m.tick() }

// Update はイベントを処理する。
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			// 終了時の保存は main 側が同期的に行う
			return m, tea.Quit
		case tea.KeyEsc:
			if m.paused || m.leveledUp {
				return m, tea.Quit
			}
			// 単語を隠して計測を止める。打ちかけのキーは捨てる
			m.paused = true
			m.engine.Reset()
			m.flash = ""
			return m, nil
		case tea.KeySpace:
			if m.leveledUp {
				// 新しいレベルの最初の単語を出題する
				m.leveledUp = false
				m.message = ""
				if err := m.newWord(time.Now()); err != nil {
					m.err = err
					return m, tea.Quit
				}
				return m, nil
			}
			if m.paused {
				m.resume(time.Now())
				return m, nil
			}
			return m, m.press(naginata.KeySpace, time.Now())
		case tea.KeyRunes:
			if m.paused || m.leveledUp {
				return m, nil
			}
			now := time.Now()
			var cmds []tea.Cmd
			for _, r := range msg.Runes {
				if key, ok := naginata.KeyFromRune(r); ok {
					if cmd := m.press(key, now); cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			}
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case tickMsg:
		now := time.Time(msg)
		if m.paused || m.leveledUp {
			return m, m.tick()
		}
		return m, tea.Batch(m.tick(), m.handleEmissions(m.engine.Flush(now), now))

	case saveErrMsg:
		m.message = "保存に失敗: " + msg.err.Error()
		return m, nil
	}
	return m, nil
}

func (m *Model) press(key naginata.Key, now time.Time) tea.Cmd {
	return m.handleEmissions(m.engine.Press(key, now), now)
}

// resume は一時停止を解除する。隠していた間に読み直しが必要なので、
// 同じ単語を最初から出題し直し、計測もやり直す。
func (m *Model) resume(now time.Time) {
	m.paused = false
	m.engine.Reset()
	m.drill.StartWord(m.drill.Word(), now)
	m.flash = ""
}

// handleEmissions は確定したかなを判定に流す。単語が終わったら次を出題する。
func (m *Model) handleEmissions(ems []naginata.Emission, now time.Time) tea.Cmd {
	for _, em := range ems {
		switch m.drill.Input(em.Text) {
		case drill.ResultAdvance:
			m.flash = ""
		case drill.ResultError:
			m.flash = "ミス: " + printable(em.Text)
		case drill.ResultWordDone:
			out := m.drill.FinishWord(now)
			m.message = resultMessage(out)
			saveCmd := m.save()
			if out.Promoted {
				// 次の単語は出さず、レベルアップ画面で Space を待つ
				m.leveledUp = true
				m.kanaAdded = out.KanaAdded
				m.engine.Reset()
				return saveCmd
			}
			if err := m.newWord(now); err != nil {
				m.err = err
				return tea.Quit
			}
			return saveCmd
		}
	}
	return nil
}

// newWord は現在のレベルに合った単語を辞書から選んで出題する。
func (m *Model) newWord(now time.Time) error {
	allowed := m.drill.Allowed()
	// 新出かなの語彙が薄いときにゲートを緩めるため、供給量を教えておく
	m.drill.SetNewKanaSupply(m.gen.CountWithUnit(allowed, m.drill.Newest()))
	word, err := m.gen.Word(allowed, m.focusUnits(), m.drill.Stage().MaxLen)
	if err != nil {
		return err
	}
	m.engine.Reset() // 前の単語の打ちかけを持ち越さない
	m.drill.StartWord(word, now)
	m.flash = ""
	return nil
}

// focusUnits は優先して出題したいかな（新出と苦手）を返す。
func (m *Model) focusUnits() []string {
	allowed := m.drill.Allowed()
	focus := append([]string{}, allowed[max(len(allowed)-2, 0):]...)
	for _, w := range m.weakItems(3) {
		focus = append(focus, w.unit)
	}
	return focus
}

// resultMessage は単語1つの結果を1行にまとめる。
func resultMessage(out drill.WordResult) string {
	switch {
	case out.Demoted:
		return fmt.Sprintf("「%s」の正答率が下がったのでレベルダウン", out.WeakUnit)
	case out.Promoted:
		return "成功率が基準を超えた! 新しいかなを追加"
	case out.Success:
		return fmt.Sprintf("成功 %.1fs / %.1fs", out.Duration.Seconds(), out.Threshold.Seconds())
	case out.Errors > 0:
		return fmt.Sprintf("失敗 (ミス %d)", out.Errors)
	default:
		return fmt.Sprintf("失敗 (時間超過 %.1fs > %.1fs)", out.Duration.Seconds(), out.Threshold.Seconds())
	}
}

// save は進捗の書き込みコマンドを返す。
// Stats はイベントループ内で更新されるため、直列化だけ同期で行い、
// ファイルI/Oは裏に逃がしてループを止めない。
func (m *Model) save() tea.Cmd {
	if m.statePath == "" {
		return nil
	}
	results, newKanaWords := m.drill.Progress()
	data, err := store.Encode(store.State{
		Level:        m.drill.Level,
		Stats:        m.drill.Stats,
		Results:      results,
		NewKanaWords: newKanaWords,
	})
	if err != nil {
		m.message = "保存に失敗: " + err.Error()
		return nil
	}
	path := m.statePath
	return func() tea.Msg {
		if err := store.Write(path, data); err != nil {
			return saveErrMsg{err: err}
		}
		return nil
	}
}

// Err は終了原因のエラーを返す。
func (m Model) Err() error { return m.err }
