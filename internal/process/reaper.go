package process

import (
	"context"
	"time"
)

func (m *Manager) StartReaper(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			m.reap(now)
		}
	}
}
func (m *Manager) reap(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, i := range m.instances {
		if !i.Mu.TryLock() {
			continue
		}
		if now.Sub(i.LastActive) > m.idleTimeout {
			_ = stopInstance(i)
			delete(m.instances, name)
		}
		i.Mu.Unlock()
	}
}
