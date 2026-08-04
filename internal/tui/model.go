// Package tui は bubbletea 製の練習画面を提供する。
package tui

import (
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

	line    lesson.Line
	width   int
	message string
	flash   string
	err     error
}

// New は画面を作り、最初の行を用意する。
func New(engine *naginata.Engine, d *drill.Drill, gen *lesson.Generator, statePath string) (Model, error) {
	m := Model{
		engine:    engine,
		drill:     d,
		gen:       gen,
		statePath: statePath,
	}
	if err := m.newLine(); err != nil {
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
		case tea.KeyCtrlC, tea.KeyEsc:
			// 終了時の保存は main 側が同期的に行う
			return m, tea.Quit
		case tea.KeyRunes:
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
		case tea.KeySpace:
			return m, m.press(naginata.KeySpace, time.Now())
		}
		return m, nil

	case tickMsg:
		now := time.Time(msg)
		return m, tea.Batch(m.tick(), m.handleEmissions(m.engine.Flush(now), now))

	case saveErrMsg:
		m.message = "保存に失敗: " + msg.err.Error()
		return m, nil
	}
	return m, nil
}

func (m *Model) press(key naginata.Key, now time.Time) tea.Cmd {
	ems := m.engine.Press(key, now)
	m.drill.MarkKeydown(now, m.engine.Presses())
	return m.handleEmissions(ems, now)
}

// handleEmissions は確定したかなを判定に流す。行が終わったら次の行を作る。
func (m *Model) handleEmissions(ems []naginata.Emission, now time.Time) tea.Cmd {
	for _, em := range ems {
		switch m.drill.Input(em.Text) {
		case drill.ResultAdvance:
			m.flash = ""
		case drill.ResultError:
			m.flash = "ミス: " + printable(em.Text)
		case drill.ResultLineDone:
			out := m.drill.FinishLine(now, m.engine.Presses())
			m.message = outcomeMessage(out, m.drill.Cfg.TargetKPM)
			saveCmd := m.save()
			if err := m.newLine(); err != nil {
				m.err = err
				return tea.Quit
			}
			return saveCmd
		}
	}
	return nil
}

// newLine は現在のレベルに合った行を辞書から組み立てる。
func (m *Model) newLine() error {
	line, err := m.gen.Generate(m.drill.Allowed(), m.focusUnits())
	if err != nil {
		return err
	}
	m.engine.Reset() // 前の行の打ちかけを持ち越さない
	m.line = line
	m.drill.StartLine(line.Units())
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

// save は進捗の書き込みコマンドを返す。
// Stats はイベントループ内で更新されるため、直列化だけ同期で行い、
// ファイルI/Oは裏に逃がしてループを止めない。
func (m *Model) save() tea.Cmd {
	if m.statePath == "" {
		return nil
	}
	data, err := store.Encode(store.State{Level: m.drill.Level, Stats: m.drill.Stats})
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
