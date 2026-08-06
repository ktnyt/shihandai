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
	interval  time.Duration // 単語と単語の間の入力を受け付けない時間

	paused    bool
	leveledUp bool // レベルアップ画面を表示中
	kanaAdded bool // 直近の昇格でかなが増えた（false なら長さの解放）
	waiting   bool // 単語間のインターバル中
	waitUntil time.Time
	width     int
	height    int
	message   string
	flash     string
	err       error

	upcoming   [][]string // 先の単語。右から左に流れてくる
	queueLevel int        // upcoming を作ったときのレベル
}

// queueLen は先読みしておく単語の数。
const queueLen = 4

// New は画面を作り、最初の単語を出題する。
// interval は単語を打ち終えてから次の単語が出るまでの間で、
// この間のキー入力は無視される（前の単語の打ち終わりの巻き込み防止）。
func New(engine *naginata.Engine, d *drill.Drill, gen *lesson.Generator, statePath string, interval time.Duration) (Model, error) {
	m := Model{
		engine:    engine,
		drill:     d,
		gen:       gen,
		statePath: statePath,
		interval:  interval,
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
		m.height = msg.Height
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
			if m.waiting {
				return m, nil
			}
			return m, m.press(naginata.KeySpace, time.Now())
		case tea.KeyRunes:
			if m.paused || m.leveledUp || m.waiting {
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
		if m.waiting {
			// インターバルが明けたら次の単語を出す
			if !now.Before(m.waitUntil) {
				m.waiting = false
				if err := m.newWord(now); err != nil {
					m.err = err
					return m, tea.Quit
				}
			}
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
	m.flash = ""
	if m.waiting {
		// 打ち終わり後のインターバル中だった場合は、単語を戻さず
		// インターバルを取り直す
		m.waitUntil = now.Add(m.interval)
		return
	}
	// 打ちかけを破棄して出し直す。ミスした位置は記録に残す
	m.drill.AbandonWord()
	m.drill.StartWord(m.drill.Word(), now)
}

// handleEmissions は確定したかなを判定に流す。単語が終わったら次を出題する。
func (m *Model) handleEmissions(ems []naginata.Emission, now time.Time) tea.Cmd {
	for _, em := range ems {
		switch m.drill.Input(em.Text, em.Keys) {
		case drill.ResultAdvance:
			m.flash = ""
		case drill.ResultError:
			m.flash = "ミス: " + printable(em.Text)
		case drill.ResultRollover:
			m.flash = "ミス: " + printable(em.Text) + " (同時押し・ノーカウント)"
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
			if m.interval > 0 {
				// 打ち終わりの巻き込みを防ぐため、少し置いてから次を出す
				m.waiting = true
				m.waitUntil = now.Add(m.interval)
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

// newWord は先読みキューの先頭を出題し、キューを補充する。
func (m *Model) newWord(now time.Time) error {
	// レベルが変わっていたら、古い条件で選んだ先読みは捨てる
	if m.queueLevel != m.drill.Level {
		m.upcoming = nil
		m.queueLevel = m.drill.Level
	}
	if err := m.fillQueue(); err != nil {
		return err
	}
	word := m.upcoming[0]
	m.upcoming = m.upcoming[1:]
	if err := m.fillQueue(); err != nil {
		return err
	}

	m.engine.Reset() // 前の単語の打ちかけを持ち越さない
	m.drill.StartWord(word, now)
	m.flash = ""
	return nil
}

// fillQueue は先読みキューを queueLen 語まで補充する。
func (m *Model) fillQueue() error {
	for len(m.upcoming) < queueLen {
		word, err := m.gen.Word(
			m.drill.Allowed(), m.drill.Newest(), m.weakUnits(3), m.drill.Stage().MaxLen)
		if err != nil {
			return err
		}
		m.upcoming = append(m.upcoming, word)
	}
	return nil
}

// weakUnits は苦手なかなを上位 n 件返す。
func (m *Model) weakUnits(n int) []string {
	var units []string
	for _, w := range m.weakItems(n) {
		units = append(units, w.unit)
	}
	return units
}

// resultMessage は単語1つの結果を1行にまとめる。
func resultMessage(out drill.WordResult) string {
	switch {
	case out.Demoted:
		return fmt.Sprintf("「%s」の正答率が下がったのでレベルダウン", out.WeakUnit)
	case out.Promoted:
		return "昇格の基準をみたした! レベルアップ"
	case out.Success:
		return fmt.Sprintf("成功 %.1fs", out.Duration.Seconds())
	default:
		return fmt.Sprintf("失敗 (ミス %d)", out.Errors)
	}
}

// save は進捗の書き込みコマンドを返す。
// Stats はイベントループ内で更新されるため、直列化だけ同期で行い、
// ファイルI/Oは裏に逃がしてループを止めない。
func (m *Model) save() tea.Cmd {
	if m.statePath == "" {
		return nil
	}
	records, newKanaWords := m.drill.Progress()
	data, err := store.Encode(store.State{
		Level:        m.drill.Level,
		Stats:        m.drill.Stats,
		Records:      records,
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
