package process

import (
	"context"
	"encoding/json"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/testutil"
	"sync"
	"testing"
	"time"
)

func TestHelper(t *testing.T) { testutil.Serve() }
func manager() *Manager {
	return New(&model.Deck{Servers: map[string]model.ServerConfig{"test": testutil.Config()}}, time.Millisecond)
}
func TestLazyReuseReapRespawn(t *testing.T) {
	m := manager()
	defer m.KillAll()
	if len(m.instances) != 0 {
		t.Fatal("spawned early")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := m.Request(ctx, "test", "tools/list", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	first, _ := m.GetOrSpawn("test")
	second, _ := m.GetOrSpawn("test")
	if first != second {
		t.Fatal("not reused")
	}
	m.reap(time.Now().Add(time.Second))
	if len(m.instances) != 0 {
		t.Fatal("not reaped")
	}
	if _, err := m.Request(ctx, "test", "tools/call", json.RawMessage(`{"name":"query"}`)); err != nil {
		t.Fatal(err)
	}
	third, _ := m.GetOrSpawn("test")
	if third == first {
		t.Fatal("not respawned")
	}
}
func TestBusyNotReapedAndConcurrentSpawn(t *testing.T) {
	m := manager()
	defer m.KillAll()
	var wg sync.WaitGroup
	got := make(chan *Instance, 8)
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			i, err := m.GetOrSpawn("test")
			if err != nil {
				t.Error(err)
				return
			}
			got <- i
		}()
	}
	wg.Wait()
	close(got)
	var first *Instance
	for i := range got {
		if first != nil && i != first {
			t.Fatal("duplicate process")
		}
		first = i
	}
	first.Mu.Lock()
	m.reap(time.Now().Add(time.Hour))
	first.Mu.Unlock()
	if len(m.instances) != 1 {
		t.Fatal("busy instance reaped")
	}
}
func TestTimeoutKillsChild(t *testing.T) {
	m := manager()
	defer m.KillAll()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := m.Request(ctx, "test", "tools/list", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	ctx2, cancel2 := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel2()
	if _, err := m.Request(ctx2, "test", "tools/call", json.RawMessage(`{"name":"hang"}`)); err == nil {
		t.Fatal("expected timeout")
	}
	i := m.instances["test"]
	select {
	case <-i.done:
	case <-time.After(time.Second):
		t.Fatal("child survived timeout")
	}
}
