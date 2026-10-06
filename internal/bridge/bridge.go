package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/process"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"io"
	"time"
)

type Bridge struct {
	deck           *model.Deck
	store          store.Store
	profile        string
	manager        *process.Manager
	RequestTimeout time.Duration
}

func New(d *model.Deck, s store.Store, profile string, idle time.Duration) (*Bridge, error) {
	if _, ok := d.Profiles[profile]; !ok {
		return nil, fmt.Errorf("unknown profile %q", profile)
	}
	if idle <= 0 {
		return nil, errors.New("idle timeout must be positive")
	}
	return &Bridge{deck: d, store: s, profile: profile, manager: process.New(d, idle), RequestTimeout: 2 * time.Minute}, nil
}
func (b *Bridge) Run(ctx context.Context, in io.Reader, out io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer b.manager.KillAll()
	go b.manager.StartReaper(ctx)
	// A reader goroutine lets signals interrupt an idle stdio connection.
	lines := make(chan []byte)
	readErrors := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(in)
		scanner.Buffer(make([]byte, 4096), process.MaxMessageSize)
		for scanner.Scan() {
			line := append([]byte(nil), scanner.Bytes()...)
			select {
			case lines <- line:
			case <-ctx.Done():
				return
			}
		}
		readErrors <- scanner.Err()
		close(lines)
	}()
	enc := json.NewEncoder(out)
	initialized, ready := false, false
	for {
		select {
		case <-ctx.Done():
			return nil
		case line, ok := <-lines:
			if !ok {
				return <-readErrors
			}
			var req Request
			resp := Response{JSONRPC: "2.0", ID: json.RawMessage(`null`)}
			if !json.Valid(line) {
				resp.Error = &Error{Code: -32700, Message: "parse error"}
			} else if err := json.Unmarshal(line, &req); err != nil || req.JSONRPC != "2.0" || req.Method == "" {
				resp.Error = &Error{Code: -32600, Message: "invalid request"}
			} else {
				if len(req.ID) == 0 {
					if req.Method == "notifications/initialized" && initialized {
						ready = true
					}
					continue
				}
				var id any
				_ = json.Unmarshal(req.ID, &id)
				switch id.(type) {
				case string, float64:
					resp.ID = req.ID
				default:
					resp.Error = &Error{Code: -32600, Message: "invalid request id"}
				}
				if resp.Error == nil {
					var result json.RawMessage
					var err error
					switch req.Method {
					case "initialize":
						var p struct {
							ProtocolVersion string `json:"protocolVersion"`
						}
						if initialized || json.Unmarshal(req.Params, &p) != nil || p.ProtocolVersion == "" {
							err = &Error{Code: -32602, Message: "invalid initialization"}
							break
						}
						version := p.ProtocolVersion
						if !process.SupportedVersion(version) {
							version = process.ProtocolVersion
						}
						result, _ = json.Marshal(map[string]any{"protocolVersion": version, "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]string{"name": "mcpdeck", "version": "0.1.0"}})
						initialized = true
					case "ping":
						result = json.RawMessage(`{}`)
					default:
						if !ready {
							err = &Error{Code: -32002, Message: "client must initialize first"}
							break
						}
						callCtx, stop := context.WithTimeout(ctx, b.RequestTimeout)
						switch req.Method {
						case "tools/list":
							var p struct {
								Cursor string `json:"cursor"`
							}
							if len(req.Params) > 0 && (json.Unmarshal(req.Params, &p) != nil || p.Cursor != "") {
								err = &Error{Code: -32602, Message: "invalid cursor"}
							} else {
								result, err = b.list(callCtx)
							}
						case "tools/call":
							result, err = b.call(callCtx, req.Params)
						default:
							err = &Error{Code: -32601, Message: "method not found"}
						}
						stop()
					}
					if err != nil {
						var rpc *Error
						if errors.As(err, &rpc) {
							resp.Error = rpc
						} else {
							resp.Error = &Error{Code: -32603, Message: "backend request failed; run mcpdeck doctor"}
						}
					} else {
						resp.Result = result
					}
				}
			}
			if err := enc.Encode(resp); err != nil {
				return err
			}
		}
	}
}
