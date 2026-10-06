package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/testutil"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHelper(t *testing.T) { testutil.Serve() }
func fixture(t *testing.T, cfg model.ServerConfig) (*Bridge, store.Store) {
	t.Helper()
	s := store.Store{Path: filepath.Join(t.TempDir(), "deck.json")}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{"test": cfg}, Profiles: map[string]model.ProfileConfig{"cursor": {TargetPath: "unused", EnabledServers: []string{"test"}}}}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	b, err := New(d, s, "cursor", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(b.manager.KillAll)
	return b, s
}
func TestCachedListNeverSpawns(t *testing.T) {
	b, _ := fixture(t, model.ServerConfig{Command: "this-executable-does-not-exist", CachedTools: []byte(`{"tools":[{"name":"query","inputSchema":{"type":"object"}}]}`)})
	raw, err := b.list(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "test__query") {
		t.Fatal(string(raw))
	}
}
func TestCorruptCacheRediscovered(t *testing.T) {
	for _, cached := range []string{
		`invalid JSON`,
		`{"tools":[{"name":42}]}`,
		`{"tools":[{}]}`,
		`{"tools":[{"name":""}]}`,
		`{"tools":[{"name":"duplicate"},{"name":"duplicate"}]}`,
		`{"tools":[{"name":"` + strings.Repeat("x", 128) + `"}]}`,
		`{"tools":[],"nextCursor":"stale"}`,
	} {
		t.Run(cached, func(t *testing.T) {
			cfg := testutil.Config()
			cfg.CachedTools = []byte(cached)
			b, s := fixture(t, cfg)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			raw, err := b.list(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var page toolPage
			if err := json.Unmarshal(raw, &page); err != nil || len(page.Tools) != 2 {
				t.Fatalf("discovery failed: %s, %v", raw, err)
			}
			d, err := s.Load()
			if err != nil {
				t.Fatal(err)
			}
			var persisted toolPage
			if err := json.Unmarshal(d.Servers["test"].CachedTools, &persisted); err != nil || len(persisted.Tools) != 2 || persisted.NextCursor != "" {
				t.Fatalf("invalid replacement cache: %+v, %v", persisted, err)
			}
		})
	}
}
func TestDiscoveryCachesPaginationAndRoutes(t *testing.T) {
	b, s := fixture(t, testutil.Config())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	raw, err := b.list(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var page toolPage
	_ = json.Unmarshal(raw, &page)
	if len(page.Tools) != 2 {
		t.Fatalf("missing pagination: %s", raw)
	}
	d, err := s.Load()
	if err != nil || len(d.Servers["test"].CachedTools) == 0 {
		t.Fatal("cache not persisted", err)
	}
	raw, err = b.call(ctx, json.RawMessage(`{"name":"test__query","arguments":{}}`))
	if err != nil || !strings.Contains(string(raw), "query") {
		t.Fatal(string(raw), err)
	}
	if _, err = b.call(ctx, json.RawMessage(`{"name":"disabled__query"}`)); err == nil {
		t.Fatal("disabled server routed")
	}
}
func TestProtocolAndIDPreservation(t *testing.T) {
	b, _ := fixture(t, model.ServerConfig{Command: "missing", CachedTools: []byte(`{"tools":[]}`)})
	input := `{"jsonrpc":"2.0","id":0,"method":"tools/list"}
{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-11-25"}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":"list","method":"tools/list"}
invalid
{"jsonrpc":"2.0","id":7,"method":"unknown"}
`
	var output bytes.Buffer
	if err := b.Run(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(&output)
	var responses []Response
	for dec.More() {
		var r Response
		if err := dec.Decode(&r); err != nil {
			t.Fatal(err)
		}
		responses = append(responses, r)
	}
	if len(responses) != 5 || responses[0].Error.Code != -32002 || string(responses[1].ID) != `"init"` || string(responses[2].ID) != `"list"` || responses[3].Error.Code != -32700 || responses[4].Error.Code != -32601 {
		t.Fatalf("unexpected responses: %+v", responses)
	}
}
