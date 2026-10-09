// Preview runs the real client facade against isolated sample files only.
// The desktop binary uses Wails bindings and does not start this HTTP server.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/altanmehmet/mcpdeck/internal/client"
)

func main() {
	port := flag.Int("port", 4174, "Loopback backend port")
	frontend := flag.Int("frontend-port", 4173, "Allowed frontend port")
	flag.Parse()
	if *port < 1024 || *port > 65535 || *frontend < 1024 || *frontend > 65535 {
		fmt.Fprintln(os.Stderr, "Invalid preview ports")
		os.Exit(1)
	}
	address := fmt.Sprintf("127.0.0.1:%d", *port)
	origin := fmt.Sprintf("http://127.0.0.1:%d", *frontend)
	app, cleanup, err := client.NewDemo()
	if err != nil {
		panic(err)
	}
	defer cleanup()
	defer app.Close()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Method != "POST" || r.Host != address || r.Header.Get("Origin") != origin || r.Header.Get("X-MCPDeck-Preview") != "1" || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "Preview request rejected", http.StatusForbidden)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 2*1024*1024)
		defer r.Body.Close()
		var req struct {
			Method string            `json:"method"`
			Args   []json.RawMessage `json:"args"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if decoder.Decode(&req) != nil {
			http.Error(w, "Invalid request", 400)
			return
		}
		var value any
		var e error
		str := func(i int) string {
			var v string
			if i < len(req.Args) {
				json.Unmarshal(req.Args[i], &v)
			}
			return v
		}
		switch req.Method {
		case "Snapshot":
			value, e = app.Snapshot()
		case "ReadPersonalDocument":
			value, e = app.ReadPersonalDocument(str(0), str(1))
		case "ReadDocument":
			value, e = app.ReadDocument(str(0), str(1))
		case "ReviewInstructions":
			var edit client.Edit
			if len(req.Args) != 1 || json.Unmarshal(req.Args[0], &edit) != nil {
				e = fmt.Errorf("Invalid edit")
			} else {
				value, e = app.ReviewInstructions(edit)
			}
		case "ApplyReview":
			value, e = app.ApplyReview(str(0))
		case "PreparePersonalReplacement":
			value, e = app.PreparePersonalReplacement(str(0), str(1), str(2))
		case "ImportShared":
			value, e = app.ImportShared(str(0))
		case "ReviewAgent":
			value, e = app.ReviewAgent(str(0), str(1), str(2))
		case "ReviewRecovery":
			value, e = app.ReviewRecovery(str(0), str(1))
		case "ReviewServer":
			value, e = app.ReviewServer(str(0), str(1))
		case "SavePlanner":
			e = fmt.Errorf("Planner settings are read-only in the isolated preview. Use the native client.")
		case "Plan", "Install", "RepairPrerequisites":
			e = fmt.Errorf("This isolated preview does not call provider accounts or run setup programs. Use the native client for installation.")
		case "Cancel":
			app.Cancel()
		default:
			e = fmt.Errorf("Unknown preview action")
		}
		out := map[string]any{"value": value}
		if e != nil {
			out["error"] = e.Error()
		}
		json.NewEncoder(w).Encode(out)
	})
	fmt.Println("Isolated client preview backend on", address)
	server := &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5_000_000_000, ReadTimeout: 10_000_000_000, WriteTimeout: 20_000_000_000, MaxHeaderBytes: 16384}
	if err = server.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
