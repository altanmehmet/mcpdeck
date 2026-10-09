package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/altanmehmet/mcpdeck/internal/install"
	"github.com/altanmehmet/mcpdeck/internal/instructions"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/testutil"
)

func TestHelper(t *testing.T) { testutil.Serve() }
func fixture(t *testing.T) (*App, *model.Deck) {
	t.Helper()
	home := t.TempDir()
	testutil.SetHome(t, home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("COPILOT_HOME", filepath.Join(home, ".copilot"))
	s := store.Store{Path: filepath.Join(home, "deck.json")}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{}, Profiles: map[string]model.ProfileConfig{}}
	for _, key := range []string{"codex", "cursor"} {
		path := filepath.Join(home, key, "mcp.json")
		if err := store.AtomicWrite(path, []byte(`{}`)); err != nil {
			t.Fatal(err)
		}
		d.Profiles[key] = model.ProfileConfig{TargetPath: path, EnabledServers: []string{}}
	}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	return New(s), d
}
func TestInstructionReviewAndStaleApproval(t *testing.T) {
	a, d := fixture(t)
	if _, err := a.manager.Apply(d, "Old shared.", false, nil); err != nil {
		t.Fatal(err)
	}
	r, err := a.ReviewInstructions(Edit{Text: "New 日本語 🚀.", Expected: "Old shared."})
	if err != nil {
		t.Fatal(err)
	}
	if current, _ := a.manager.Load(); current != "Old shared." {
		t.Fatal("review wrote state")
	}
	if _, err = a.ApplyReview("forged"); err == nil {
		t.Fatal("forged review accepted")
	}
	if _, err = a.ApplyReview(r.Token); err != nil {
		t.Fatal(err)
	}
	if current, _ := a.manager.Load(); current != "New 日本語 🚀." {
		t.Fatal("apply failed")
	}
	if _, err = a.ApplyReview(r.Token); err == nil {
		t.Fatal("review could be replayed")
	}
	r, err = a.ReviewInstructions(Edit{Text: "Next.", Expected: "New 日本語 🚀."})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.manager.Apply(d, "Changed in terminal.", false, nil); err != nil {
		t.Fatal(err)
	}
	if _, err = a.ApplyReview(r.Token); err == nil {
		t.Fatal("stale source overwritten")
	}
	if current, _ := a.manager.Load(); current != "Changed in terminal." {
		t.Fatal("stale write changed source")
	}
	r, err = a.ReviewInstructions(Edit{Text: "Never applied.", Expected: "Changed in terminal."})
	if err != nil {
		t.Fatal(err)
	}
	a.pending.expires = time.Now().Add(-time.Second)
	if _, err = a.ApplyReview(r.Token); err == nil {
		t.Fatal("expired review applied")
	}
}
func TestPersonalEditorHidesAndPreservesSharedAndMetadata(t *testing.T) {
	a, d := fixture(t)
	if _, err := a.manager.Apply(d, "Shared stays.", false, nil); err != nil {
		t.Fatal(err)
	}
	doc, err := a.manager.SelectDocument(d, "cursor", "")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := a.manager.ReadDocument(d, doc)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.AtomicWrite(doc.Path, []byte(raw+"\nOriginal personal.")); err != nil {
		t.Fatal(err)
	}
	view, err := a.ReadPersonalDocument("cursor", doc.Path)
	if err != nil || view.Text != "Original personal." {
		t.Fatal("editor exposed internal rule metadata or shared block", view, err)
	}
	r, err := a.ReviewInstructions(Edit{Agent: "cursor", Path: doc.Path, Personal: true, Text: "", Expected: view.Expected})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(r.Current, "mcpdeck:") || r.Proposed != "" {
		t.Fatal("review leaked internal markers")
	}
	if _, err = a.ApplyReview(r.Token); err != nil {
		t.Fatal(err)
	}
	updated, err := a.manager.ReadDocument(d, doc)
	if err != nil || !strings.Contains(updated, "alwaysApply: true") || !strings.Contains(updated, "Shared stays.") || strings.Contains(updated, "Original personal.") {
		t.Fatal("personal clear lost protected text", err)
	}
	if central, _ := a.manager.Load(); central != "Shared stays." {
		t.Fatal("personal edit changed shared source")
	}
	testutil.AssertPrivateFile(t, doc.Path)
}
func TestSnapshotNeverExportsServerCredentials(t *testing.T) {
	a, d := fixture(t)
	d.Servers["private-server"] = model.ServerConfig{Command: "example", Args: []string{"password=never-export"}, Env: map[string]string{"SECRET": "never-export"}, Variables: map[string]string{"TOKEN": "never-export"}}
	if err := a.s.Save(d); err != nil {
		t.Fatal(err)
	}
	state, err := a.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(state)
	if strings.Contains(string(raw), "never-export") {
		t.Fatal("credentials escaped through server inventory")
	}
}
func TestReviewedServerActivationRejectsChangedDeck(t *testing.T) {
	a, d := fixture(t)
	d.Servers["fixture"] = testutil.Config()
	if err := a.s.Save(d); err != nil {
		t.Fatal(err)
	}
	r, err := a.ReviewServer("fixture", "enable")
	if err != nil {
		t.Fatal(err)
	}
	d.Profiles["cursor"] = model.ProfileConfig{TargetPath: filepath.Join(t.TempDir(), "other.json")}
	if err = a.s.Save(d); err != nil {
		t.Fatal(err)
	}
	if _, err = a.ApplyReview(r.Token); err == nil {
		t.Fatal("review applied to different target paths")
	}
	fresh, _ := a.s.Load()
	if fresh.IsEnabled("codex", "fixture") {
		t.Fatal("changed deck was mutated")
	}
}
func TestInstallUsesReviewedRecipeAndPrivateChannel(t *testing.T) {
	a, d := fixture(t)
	cfg := testutil.Config()
	cfg.Variables = nil
	cfg.Env["PRIVATE_TOKEN"] = "${PRIVATE_TOKEN}"
	raw, _ := json.Marshal(cfg)
	p := install.Plan{Version: 1, Name: "fixture", Connection: string(raw)}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	view := PlanView{ID: token(), Name: p.Name, Required: []string{"PRIVATE_TOKEN"}, Targets: installationTargets(d)}
	a.plan = &planned{recipe: p, view: view, deckHash: deckHash(d)}
	if _, err := a.Install("forged", false, map[string]string{}); err == nil {
		t.Fatal("unknown recipe accepted")
	}
	id, err := a.Install(view.ID, false, map[string]string{"PRIVATE_TOKEN": "unpredictable-private-proof"})
	if err != nil {
		t.Fatal(err)
	}
	var status job
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		status, err = a.Job(id)
		if err != nil {
			t.Fatal(err)
		}
		if status.Status != "running" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if status.Status != "done" {
		t.Fatal("fixture install failed", status.Status, status.Message)
	}
	serialized, _ := json.Marshal(status)
	if strings.Contains(string(serialized), "unpredictable-private-proof") {
		t.Fatal("private value escaped into job status")
	}
	fresh, _ := a.s.Load()
	if !fresh.IsEnabled("codex", "fixture") || !fresh.IsEnabled("cursor", "fixture") {
		t.Fatal("verified MCP did not reach every reviewed target")
	}
	for _, profile := range fresh.Profiles {
		raw, err = os.ReadFile(profile.TargetPath)
		if err != nil || !strings.Contains(string(raw), "fixture") {
			t.Fatal("target config missing MCP", err)
		}
	}
	testutil.AssertPrivateFile(t, a.s.Path)
}
func TestSharedClearKeepsPersonalFiles(t *testing.T) {
	a, d := fixture(t)
	if _, err := a.manager.Apply(d, "Shared", false, nil); err != nil {
		t.Fatal(err)
	}
	doc, _ := a.manager.SelectDocument(d, "codex", "")
	raw, _ := a.manager.ReadDocument(d, doc)
	store.AtomicWrite(doc.Path, []byte("Personal\n"+raw))
	r, err := a.ReviewInstructions(Edit{Text: "", Expected: "Shared"})
	if err != nil || !strings.Contains(r.Detail, "Remove only") {
		t.Fatal("clear not explicit", err)
	}
	if _, err = a.ApplyReview(r.Token); err != nil {
		t.Fatal(err)
	}
	raw, _ = a.manager.ReadDocument(d, doc)
	if !strings.Contains(raw, "Personal") || strings.Contains(raw, "mcpdeck:global-instructions") {
		t.Fatal("clear removed personal text")
	}
}

var _ = instructions.MaxBytes

func TestCurrentAgentInstructionsIncludeSharedWithoutChangingPersonalEditor(t *testing.T) {
	a, d := fixture(t)
	if _, err := a.manager.Apply(d, "Shared current guidance.", false, nil); err != nil {
		t.Fatal(err)
	}
	doc, err := a.manager.SelectDocument(d, "codex", "")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := a.manager.ReadDocument(d, doc)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.AtomicWrite(doc.Path, []byte("Personal current guidance.\n"+raw)); err != nil {
		t.Fatal(err)
	}
	view, err := a.ReadPersonalDocument("codex", doc.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(view.Current, "Shared current guidance.") || !strings.Contains(view.Current, "Personal current guidance.") {
		t.Fatal("saved guidance hidden")
	}
	if view.Text != "Personal current guidance." || strings.Contains(view.Current, "mcpdeck:global-instructions") {
		t.Fatal("editor protections or presentation changed")
	}
}

func TestInstructionFileMetadataAndSharedAgentReview(t *testing.T) {
	a, d := fixture(t)
	d.Profiles["gemini-cli"] = model.ProfileConfig{TargetPath: filepath.Join(a.manager.Home, "gemini-settings.json")}
	d.Profiles["antigravity"] = model.ProfileConfig{TargetPath: filepath.Join(a.manager.Home, "antigravity-settings.json")}
	if err := a.s.Save(d); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(a.manager.Home, ".gemini", "GEMINI.md")
	state, err := a.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	missing := false
	for _, item := range state.InstructionFiles {
		if item.Path == path && item.Status == "not created" {
			missing = true
		}
	}
	if !missing {
		t.Fatal("missing instruction file not identified")
	}
	if err := store.AtomicWrite(path, []byte("Shared physical personal file.")); err != nil {
		t.Fatal(err)
	}
	review, err := a.ReviewInstructions(Edit{Agent: "gemini-cli", Path: path, Personal: true, Text: "Updated physical file.", Expected: "Shared physical personal file."})
	if err != nil {
		t.Fatal(err)
	}
	if len(review.Results) != 2 {
		t.Fatal("review hides affected aliases", review.Results)
	}
	if _, err = a.ApplyReview(review.Token); err != nil {
		t.Fatal(err)
	}
	for _, agent := range []string{"gemini-cli", "antigravity"} {
		view, err := a.ReadPersonalDocument(agent, path)
		if err != nil || view.Current != "Updated physical file." {
			t.Fatal(agent, view, err)
		}
	}
}
