package instructions

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestExistingGlobalRuleDiscoveryAndAgentOnlyEdit(t *testing.T) {
	m, d := fixture(t)
	path := filepath.Join(m.Home, ".cursor", "rules", "personal", "style.mdc")
	write(t, path, "---\nalwaysApply: true\n---\nUse existing project conventions.\n")
	project := filepath.Join(m.Home, "project", ".cursor", "rules", "private.mdc")
	write(t, project, "Project-specific only")
	docs, err := m.Documents(d)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, doc := range docs {
		if doc.Path == project {
			t.Fatal("project rules scanned")
		}
		if doc.Agent == "cursor" && doc.Path == path {
			found = true
		}
	}
	if !found {
		t.Fatal("non-MCPDeck global rule not discovered")
	}
	doc, err := m.SelectDocument(d, "cursor", path)
	if err != nil {
		t.Fatal(err)
	}
	current, err := m.ReadDocument(d, doc)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.ReplaceAll(current, "Use existing project conventions.", "Follow existing naming patterns.")
	if err := m.SaveDocument(d, doc, updated, current); err != nil {
		t.Fatal(err)
	}
	if read(t, path) != updated || read(t, path+".mcpdeck-backup") != current {
		t.Fatal("native file edit/backup failed")
	}
	if read(t, project) != "Project-specific only" {
		t.Fatal("project file changed")
	}
	if _, err := m.ReadDocument(d, Document{Agent: "cursor", Path: project}); err == nil {
		t.Fatal("unregistered project file accepted")
	}
	if err := m.SaveDocument(d, doc, "stale edit", current); err == nil {
		t.Fatal("stale file edit overwrote a newer change")
	}
}

func TestNativeEditProtectsSharedBlock(t *testing.T) {
	m, d := fixture(t)
	doc, err := m.SelectDocument(d, "codex", "")
	if err != nil {
		t.Fatal(err)
	}
	write(t, doc.Path, "Personal: use tabs.\n")
	if _, err := m.Apply(d, "Shared: never log secrets.", false, nil); err != nil {
		t.Fatal(err)
	}
	current, err := m.ReadDocument(d, doc)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.ReplaceAll(current, "use tabs", "use project formatting")
	if err := m.SaveDocument(d, doc, updated, current); err != nil {
		t.Fatal(err)
	}
	if err := m.SaveDocument(d, doc, strings.ReplaceAll(updated, "never log secrets", "log secrets"), updated); err == nil {
		t.Fatal("shared guidance silently changed in one agent")
	}
	if !strings.Contains(read(t, doc.Path), "Shared: never log secrets.") {
		t.Fatal("shared guidance lost")
	}
	text, _ := m.Load()
	if text != "Shared: never log secrets." {
		t.Fatal("native edit changed canonical guidance")
	}
}

func TestOneInstructionAppendIsIdempotentAndKeepsExistingText(t *testing.T) {
	text, err := AppendText("Existing shared guidance.", "Never log secrets.")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Existing shared guidance.") {
		t.Fatal("append removed earlier instructions")
	}
	again, err := AppendText(text, "Never log secrets.")
	if err != nil || again != text {
		t.Fatal("retry duplicated the instruction", err)
	}
	if _, err := AppendText(text, " "); err == nil {
		t.Fatal("empty instruction accepted")
	}
	shared, err := SharedText("---\nalwaysApply: true\n---\nExisting personal text.\n" + begin + "\nExisting shared text.\n" + end)
	if err != nil || shared != "Existing personal text.\n\nExisting shared text." {
		t.Fatalf("imported guidance = %q, %v", shared, err)
	}
}

func TestModularPersonalRulesAreVisible(t *testing.T) {
	m, d := fixture(t)
	claudePath := filepath.Join(m.Home, ".claude", "rules", "security.md")
	copilotPath := filepath.Join(m.CopilotHome, "instructions", "personal", "formatting.instructions.md")
	write(t, claudePath, "Never expose credentials.")
	write(t, copilotPath, "---\napplyTo: '**'\n---\nRespect existing formatting.")
	for _, item := range []Document{{Agent: "claude-code", Path: claudePath}, {Agent: "copilot-cli", Path: copilotPath}} {
		text, err := m.ReadDocument(d, item)
		if err != nil || text == "" {
			t.Fatalf("modular personal rule not readable: %+v, %v", item, err)
		}
	}
}

func TestReplacePersonalTextKeepsExactManagedBlock(t *testing.T) {
	block := begin + "\nShared 日本語.\n" + end
	current := "Before.\n" + block + "\nAfter."
	for _, replacement := range []string{"", "New personal 🚀."} {
		updated, err := ReplacePersonalText(current, replacement)
		if err != nil {
			t.Fatal(err)
		}
		got, err := managedBlock(updated)
		if err != nil || got != block || strings.Contains(updated, "Before.") || strings.Contains(updated, "After.") {
			t.Fatal("personal replacement changed shared block", err)
		}
	}
	if _, err := ReplacePersonalText(begin+"\nBroken", "New"); err == nil {
		t.Fatal("malformed block accepted")
	}
	if _, err := ReplacePersonalText(current, block); err == nil {
		t.Fatal("replacement accepted managed markers")
	}
	if text, err := ReplacePersonalText("Unmanaged", ""); err != nil || text != "" {
		t.Fatal("unmanaged file could not be cleared", err)
	}
}

func TestReplacePersonalTextPreservesRuleActivationMetadata(t *testing.T) {
	header := "---\nalwaysApply: true\ndescription: Existing rule\n---\n"
	block := begin + "\nShared.\n" + end
	for _, suffix := range []string{"", "\n" + block + "\n"} {
		for _, replacement := range []string{"", "New instructions."} {
			updated, err := ReplacePersonalText(header+"Old instructions."+suffix, replacement)
			if err != nil || !strings.HasPrefix(updated, header) || strings.Contains(updated, "Old instructions.") {
				t.Fatal("replacement removed rule metadata", err)
			}
		}
	}
}
