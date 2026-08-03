package sentence

import (
	"context"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/ktnyt/shihandai/internal/curriculum"
)

// TestOllamaLive は実際の Ollama に接続して生成を確かめる。
// SHIHANDAI_LIVE=1 のときだけ動く。
func TestOllamaLive(t *testing.T) {
	if os.Getenv("SHIHANDAI_LIVE") != "1" {
		t.Skip("SHIHANDAI_LIVE=1 のときだけ動く")
	}
	model := os.Getenv("SHIHANDAI_MODEL")
	if model == "" {
		model = "hf.co/LiquidAI/LFM2.5-8B-A1B-GGUF:Q6_K"
	}

	ollama := NewOllama("http://localhost:11434", model)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := ollama.Available(ctx); err != nil {
		t.Fatal(err)
	}

	g := &Generator{LLM: ollama, Cfg: DefaultConfig(), Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}
	for _, level := range []int{1, 15, 40} {
		allowed := curriculum.For(level)
		start := time.Now()
		line, err := g.Generate(ctx, allowed)
		if err != nil {
			t.Fatalf("レベル %d: %v", level, err)
		}
		t.Logf("レベル %d (%s, %v): %s", level, line.Source, time.Since(start).Round(time.Second), line.Text())
		if _, ok := Segment(line.Text(), allowed); !ok {
			t.Errorf("レベル %d の例文に使えないかなが含まれる: %q", level, line.Text())
		}
	}
}
