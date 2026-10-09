package instructions

import (
	"path/filepath"
	"runtime"
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

func TestPersonalTextEditorRoundTripPreservesProtectedSections(t *testing.T) {
	header := "---\nalwaysApply: true\n---\n"
	block := begin + "\nShared stays.\n" + end
	raw := header + "Personal before.\n\n" + block + "\nPersonal after.\n"
	personal, err := PersonalText(raw)
	if err != nil || strings.Contains(personal, "Shared stays") || strings.Contains(personal, "alwaysApply") || !strings.Contains(personal, "Personal after.") {
		t.Fatal(personal, err)
	}
	updated, err := ReplacePersonalText(raw, personal+"\nNew rule.")
	if err != nil || !strings.HasPrefix(updated, header) || !strings.Contains(updated, block) {
		t.Fatal("round trip changed protected content", err)
	}
	if _, err = PersonalText(begin + "\nBroken"); err == nil {
		t.Fatal("malformed managed block accepted")
	}
}

func TestCurrentTextSupportsLargeExistingDocuments(t *testing.T) {
	raw := "---\nalwaysApply: true\n---\n" + strings.Repeat("Existing personal rule.\n", 2000) + begin + "\nShared guidance.\n" + end
	current, err := CurrentText(raw)
	if err != nil || !strings.Contains(current, "Shared guidance.") || strings.Contains(current, "alwaysApply") || strings.Contains(current, begin) {
		t.Fatal("current document display invalid", err)
	}
	if len(current) <= MaxBytes {
		t.Fatal("fixture did not exercise large document")
	}
}

func TestAntigravityGlobalAlternativesAndFlatRuleDiscovery(t *testing.T) {
	m, d := fixture(t)
	for _, path := range []string{filepath.Join(m.Home, ".gemini", "AGENTS.md"), filepath.Join(m.Home, ".gemini", "config", "GEMINI.md"), filepath.Join(m.Home, ".gemini", "config", "rules", "security.md")} {
		write(t, path, "Current global guidance.")
	}
	nested := filepath.Join(m.Home, ".gemini", "config", "rules", "nested", "ignored.md")
	write(t, nested, "Not a flat global rule.")
	docs, err := m.Documents(d)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, doc := range docs {
		if doc.Agent == "antigravity" {
			seen[doc.Path] = true
		}
	}
	if !seen[filepath.Join(m.Home, ".gemini", "AGENTS.md")] || !seen[filepath.Join(m.Home, ".gemini", "config", "rules", "security.md")] || seen[nested] {
		t.Fatal("wrong global rule scope", seen)
	}
}

func TestGeminiConfiguredMarkdownContextIsInventoried(t *testing.T) {
	m, d := fixture(t)
	write(t, filepath.Join(m.Home, ".gemini", "settings.json"), `{"context":{"fileName":["AGENTS.md","../project.md","oauth_creds.json"]}}`)
	agentFile := filepath.Join(m.Home, ".gemini", "AGENTS.md")
	write(t, agentFile, "Configured global context.")
	write(t, filepath.Join(m.Home, ".gemini", "oauth_creds.json"), "not an instruction file")
	docs, err := m.Documents(d)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, doc := range docs {
		if doc.Agent == "gemini-cli" {
			if doc.Path == agentFile {
				found = true
			}
			if strings.HasSuffix(doc.Path, "oauth_creds.json") {
				t.Fatal("non-Markdown settings exposed")
			}
		}
	}
	if !found {
		t.Fatal("configured context filename not discovered")
	}
}

func TestCopilotLocalProfileInstructionsAreVisibleButPromptsExcluded(t *testing.T) {
	m, d := fixture(t)
	user := filepath.Join(m.Home, ".config", "Code", "User")
	if runtime.GOOS == "darwin" {
		user = filepath.Join(m.Home, "Library", "Application Support", "Code", "User")
	}
	if runtime.GOOS == "windows" {
		appData := filepath.Join(m.Home, "AppData", "Roaming")
		t.Setenv("APPDATA", appData)
		user = filepath.Join(appData, "Code", "User")
	}
	rule := filepath.Join(user, "profiles", "profile-one", "prompts", "security.instructions.md")
	prompt := filepath.Join(user, "prompts", "work.prompt.md")
	write(t, rule, "---\napplyTo: '**'\n---\nCurrent Local profile guidance.")
	write(t, prompt, "This is a task prompt, not a global rule.")
	docs, err := m.Documents(d)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, doc := range docs {
		if doc.Agent == "copilot" {
			if doc.Path == rule {
				found = true
			}
			if doc.Path == prompt {
				t.Fatal("prompt was shown as an instruction")
			}
		}
	}
	if !found {
		t.Fatal("VS Code Local profile rules not inventoried")
	}
}

func TestSelectDocumentDoesNotHideModularGuidance(t *testing.T) {
	m, d := fixture(t)
	primary := filepath.Join(m.CopilotHome, "copilot-instructions.md")
	rule := filepath.Join(m.CopilotHome, "instructions", "style.instructions.md")
	write(t, rule, "Use existing conventions.")
	ignored := filepath.Join(m.CopilotHome, "instructions", "notes.md")
	write(t, ignored, "Not a Copilot instruction file.")
	for _, empty := range []bool{false, true} {
		if empty {
			write(t, primary, "\n")
		}
		doc, err := m.SelectDocument(d, "copilot-cli", "")
		if err != nil || doc.Path != rule {
			t.Fatal("modular guidance hidden", doc, err)
		}
	}
	docs, err := m.Documents(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, doc := range docs {
		if doc.Path == ignored {
			t.Fatal("unloaded Markdown listed as instructions")
		}
	}
	write(t, filepath.Join(m.CopilotHome, "instructions", "security.instructions.md"), "Never log credentials.")
	if _, err := m.SelectDocument(d, "copilot-cli", ""); err == nil {
		t.Fatal("ambiguous modular selection must require --path")
	}
	doc, err := m.SelectDocument(d, "copilot-cli", primary)
	if err != nil || doc.Path != primary {
		t.Fatal("explicit path changed", err)
	}
	write(t, primary, "Main guidance.")
	doc, err = m.SelectDocument(d, "copilot-cli", "")
	if err != nil || doc.Path != primary {
		t.Fatal("established populated main file lost priority", err)
	}
}
