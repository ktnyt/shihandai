package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ktnyt/shihandai/internal/drill"
)

func TestLoadMissingFileReturnsInitialState(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "none.json"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Level != 1 || s.Stats == nil {
		t.Fatalf("初期状態でない: %+v", s)
	}
}

func TestSaveAndLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	want := State{
		Level: 7,
		Stats: map[string]*drill.UnitStat{
			"あ": {Attempts: 10, Errors: 2, Recent: []bool{true, false, true}},
		},
	}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Level != want.Level {
		t.Errorf("Level = %d, want %d", got.Level, want.Level)
	}
	s := got.Stats["あ"]
	if s == nil || s.Attempts != 10 || s.Errors != 2 || len(s.Recent) != 3 {
		t.Errorf("Stats[あ] = %+v", s)
	}
}

func TestLoadDropsNullStats(t *testing.T) {
	// 手で編集されたファイルの null 値で後段が panic しないための回帰テスト
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"level":3,"stats":{"あ":null,"い":{"attempts":1}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Stats["あ"]; ok {
		t.Errorf("null の統計が残っている")
	}
	if got.Stats["い"] == nil {
		t.Errorf("正常な統計が消えている")
	}

	// 読み込んだ状態でそのまま練習しても落ちない
	d := drill.New(drill.DefaultConfig(), got.Level, got.Stats)
	d.StartWord([]string{"あ"}, time.Unix(0, 0))
	if res := d.Input("あ", 0); res != drill.ResultWordDone {
		t.Errorf("Input = %v", res)
	}
}

func TestLoadInvalidLevelClamped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"level":0}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Level != 1 {
		t.Errorf("Level = %d, want 1", got.Level)
	}
}

func TestWriteLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := Save(path, State{Level: 1, Stats: map[string]*drill.UnitStat{}}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "state.json" {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("一時ファイルが残っている: %v", names)
	}
}
