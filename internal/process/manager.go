package process

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/google/uuid"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

const ProtocolVersion = "2025-11-25"
const MaxMessageSize = 16 * 1024 * 1024

func SupportedVersion(v string) bool {
	return v == ProtocolVersion || v == "2025-06-18" || v == "2025-03-26" || v == "2024-11-05"
}

type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *RPCError) Error() string { return e.Message }

type wire struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}
type Instance struct {
	ServerName  string
	Cmd         *exec.Cmd
	Stdin       io.WriteCloser
	Stdout      io.ReadCloser
	LastActive  time.Time
	Mu          sync.Mutex // Serializes requests and prevents reaping an in-flight request.
	done        chan struct{}
	messages    chan wire
	readDone    chan struct{}
	writeMu     sync.Mutex
	initialized bool
}
type Manager struct {
	mu          sync.RWMutex
	instances   map[string]*Instance
	deck        *model.Deck
	idleTimeout time.Duration
	closed      bool
}

func New(d *model.Deck, idle time.Duration) *Manager {
	return &Manager{instances: map[string]*Instance{}, deck: d, idleTimeout: idle}
}
func (m *Manager) GetOrSpawn(name string) (*Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, errors.New("process manager is closed")
	}
	if inst := m.instances[name]; inst != nil {
		select {
		case <-inst.done:
			delete(m.instances, name)
		default:
			return inst, nil
		}
	}
	cfg, ok := m.deck.Servers[name]
	if !ok {
		return nil, fmt.Errorf("unknown server %s", name)
	}
	cfg, err := model.Resolve(cfg)
	if err != nil {
		return nil, err
	}
	cfg = model.Stdio(cfg)
	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Env = os.Environ()
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	configureProcess(cmd)
	// Child stderr is discarded because third-party programs may print credentials.
	cmd.Stderr = io.Discard
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, err
	}
	if err = startProcess(cmd); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("start %s: %w", name, err)
	}
	inst := &Instance{ServerName: name, Cmd: cmd, Stdin: stdin, Stdout: stdout, LastActive: time.Now(), done: make(chan struct{}), messages: make(chan wire, 32), readDone: make(chan struct{})}
	m.instances[name] = inst
	go inst.read()
	go func() { _ = cmd.Wait(); releaseProcess(cmd); close(inst.done); stdin.Close(); stdout.Close() }()
	return inst, nil
}
func (i *Instance) read() {
	defer close(i.readDone)
	defer close(i.messages)
	scanner := bufio.NewScanner(i.Stdout)
	scanner.Buffer(make([]byte, 4096), MaxMessageSize)
	for scanner.Scan() {
		var w wire
		if json.Unmarshal(scanner.Bytes(), &w) != nil || w.JSONRPC != "2.0" {
			return
		}
		select {
		case i.messages <- w:
		case <-i.done:
			return
		}
	}
}
func (i *Instance) write(w wire) error {
	i.writeMu.Lock()
	defer i.writeMu.Unlock()
	return json.NewEncoder(i.Stdin).Encode(w)
}
func (i *Instance) call(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
	id, _ := json.Marshal(uuid.NewString())
	if err := i.write(wire{JSONRPC: "2.0", ID: id, Method: method, Params: params}); err != nil {
		return nil, err
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case w, ok := <-i.messages:
			if !ok {
				return nil, errors.New("child connection closed")
			}
			if w.Method != "" {
				if len(w.ID) > 0 {
					reply := wire{JSONRPC: "2.0", ID: w.ID, Error: &RPCError{Code: -32601, Message: "client capability not supported"}}
					if w.Method == "ping" {
						reply.Error = nil
						reply.Result = json.RawMessage(`{}`)
					}
					if err := i.write(reply); err != nil {
						return nil, err
					}
				}
				continue
			}
			if string(w.ID) != string(id) {
				continue
			}
			if w.Error != nil {
				return nil, w.Error
			}
			if w.Result == nil {
				return nil, errors.New("child response has no result")
			}
			return w.Result, nil
		}
	}
}

// Request holds the instance lock from initialization through the final response.
func (m *Manager) Request(ctx context.Context, name, method string, params json.RawMessage) (json.RawMessage, error) {
	inst, err := m.GetOrSpawn(name)
	if err != nil {
		return nil, err
	}
	inst.Mu.Lock()
	defer inst.Mu.Unlock()
	select {
	case <-inst.done:
		return nil, errors.New("child exited before request")
	default:
	}
	inst.LastActive = time.Now()
	defer func() { inst.LastActive = time.Now() }()
	// Closing pipes on timeout also releases a blocked write to an unresponsive child.
	stop := context.AfterFunc(ctx, func() { inst.Stdin.Close(); inst.Stdout.Close(); _ = forceProcess(inst.Cmd) })
	defer stop()
	if !inst.initialized {
		p, _ := json.Marshal(map[string]any{"protocolVersion": ProtocolVersion, "capabilities": map[string]any{}, "clientInfo": map[string]string{"name": "mcpdeck", "version": "0.1.0"}})
		result, e := inst.call(ctx, "initialize", p)
		if e != nil {
			_ = forceProcess(inst.Cmd)
			return nil, e
		}
		var init struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if json.Unmarshal(result, &init) != nil || !SupportedVersion(init.ProtocolVersion) {
			_ = forceProcess(inst.Cmd)
			return nil, errors.New("unsupported child protocol version")
		}
		if e = inst.write(wire{JSONRPC: "2.0", Method: "notifications/initialized"}); e != nil {
			return nil, e
		}
		inst.initialized = true
	}
	result, err := inst.call(ctx, method, params)
	if err != nil {
		var rpc *RPCError
		if !errors.As(err, &rpc) {
			_ = forceProcess(inst.Cmd)
		}
	}
	return result, err
}
func (m *Manager) Touch(name string) {
	m.mu.RLock()
	i := m.instances[name]
	m.mu.RUnlock()
	if i != nil {
		i.Mu.Lock()
		i.LastActive = time.Now()
		i.Mu.Unlock()
	}
}
func stopInstance(i *Instance) error {
	i.Stdin.Close()
	_ = terminateProcess(i.Cmd)
	select {
	case <-i.done:
	case <-time.After(2 * time.Second):
		_ = forceProcess(i.Cmd)
		select {
		case <-i.done:
		case <-time.After(2 * time.Second):
			return errors.New("child did not exit")
		}
	}
	// Kill remaining descendants in the process group even if the launcher already exited.
	_ = forceProcess(i.Cmd)
	i.Stdout.Close()
	return nil
}
func (m *Manager) Kill(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	i := m.instances[name]
	if i == nil {
		return nil
	}
	i.Mu.Lock()
	defer i.Mu.Unlock()
	err := stopInstance(i)
	delete(m.instances, name)
	return err
}
func (m *Manager) KillAll() {
	m.mu.Lock()
	m.closed = true
	instances := m.instances
	m.instances = map[string]*Instance{}
	m.mu.Unlock()
	var wg sync.WaitGroup
	for _, i := range instances {
		wg.Add(1)
		go func(i *Instance) { defer wg.Done(); _ = stopInstance(i) }(i)
	}
	wg.Wait()
}
