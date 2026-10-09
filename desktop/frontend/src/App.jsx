import { useEffect, useRef, useState } from "react";
import {
  Database,
  Plus,
  FileText,
  Users,
  GearSix,
  ArrowClockwise,
  ArrowCounterClockwise,
  MagnifyingGlass,
  ArrowRight,
  CheckCircle,
  X,
  WarningCircle,
  PaperPlaneRight,
  Stop,
  CaretDown,
} from "@phosphor-icons/react";
import { call, friendly, isNative } from "./api";
import { lineDiff } from "./diff";
import { groupInstructionFiles, preferredInstructionFile } from "./instruction-files";

const initialAddition =
  "- Verify changes before reporting success.\n- Ask before making a breaking change.";
const navigation = [
  ["servers", "MCP Servers", Database],
  ["new", "New MCP", Plus],
  ["instructions", "Instructions", FileText],
  ["agents", "Agents", Users],
];
const automatic = (rows) =>
  rows.filter((row) => !["manual", "not detected"].includes(row.Status));
const changed = (rows) =>
  automatic(rows).filter(
    (row) => !["unchanged", "synced", "already synced"].includes(row.Status),
  );
function Button({ children, primary = false, danger = false, ...props }) {
  return (
    <button
      className={`button ${primary ? "primary" : ""} ${danger ? "danger" : ""}`}
      {...props}
    >
      {children}
    </button>
  );
}
function ErrorText({ text }) {
  return text ? (
    <div className="error-message" role="alert">
      <WarningCircle size={20} />
      <span>{text}</span>
    </div>
  ) : null;
}
function DiffPane({ title, items, added }) {
  return (
    <section className="diff-pane">
      <h2>{title}</h2>
      <pre>
        {items.map((item, index) => (
          <span
            key={index}
            className={item.changed ? (added ? "addition" : "deletion") : ""}
          >
            {item.changed ? (added ? "+ " : "− ") : ""}
            {(item.changed ? item.line.replace(/^- /, "") : item.line) || " "}
            <br />
          </span>
        ))}
      </pre>
    </section>
  );
}
function Results({ rows }) {
  return (
    <div className="table-scroll">
      <table>
        <thead>
          <tr>
            <th>Agent</th>
            <th>Change</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={r.Agent + r.Path + i}>
              <td>{friendly(r.Agent)}</td>
              <td>
                {["pending", "synced"].includes(r.Status)
                  ? "Update shared section"
                  : r.Status === "update this file"
                    ? "Update personal text"
                    : r.Status === "cleared"
                      ? "Remove shared section"
                      : r.Status}
                {r.Status === "manual" && <small>{r.Detail}</small>}
                {r.Status === "failed" && <small>{r.Detail}</small>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
function Dialog({ title, children, onClose, footer, busy = false }) {
  const ref = useRef(null);
  useEffect(() => {
    const before = document.activeElement;
    if (!ref.current.open) ref.current.showModal();
    return () => before?.focus?.();
  }, []);
  return (
    <dialog
      ref={ref}
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
      onClick={(event) => {
        if (!busy && event.target === ref.current) onClose();
      }}
    >
      <header>
        <h2>{title}</h2>
        <button
          className="icon-button"
          aria-label="Close dialog"
          disabled={busy}
          onClick={onClose}
        >
          <X size={22} />
        </button>
      </header>
      {children}
      <footer>{footer}</footer>
    </dialog>
  );
}
export function App() {
  const [state, setState] = useState(null),
    [section, setSection] = useState("servers"),
    [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [busy, setBusy] = useState(false);
  const [text, setText] = useState(""),
    [baseline, setBaseline] = useState(""),
    [expected, setExpected] = useState(""),
    [file, setFile] = useState(null),
    [tab, setTab] = useState("edit"),
    [mode, setMode] = useState("replace"),
    [review, setReview] = useState(null),
    [savedResults, setSavedResults] = useState([]),
    [query, setQuery] = useState(""),
 [currentInstructions,setCurrentInstructions] = useState(""),
 [instructionAgent,setInstructionAgent] = useState("");
  const [dialog, setDialog] = useState(null),
    [instruction, setInstruction] = useState(""),
    [serverReview, setServerReview] = useState(null),
    [planner, setPlanner] = useState({ provider: "codex" }),
 [plannerBaseline,setPlannerBaseline] = useState(null),
 [agentFilter,setAgentFilter] = useState("");
  const [request, setRequest] = useState(""),
    [messages, setMessages] = useState([]),
    [plan, setPlan] = useState(null),
    [job, setJob] = useState(null),
    [privateValues, setPrivateValues] = useState({}),
    [manual, setManual] = useState(false),
    [showPlan, setShowPlan] = useState(false);
  const editRef = useRef(null),
    jobTimer = useRef(null),
    initialized = useRef(false),
    [elapsed, setElapsed] = useState(0);
  const dirty = mode === "append" ? Boolean(text.trim()) : text !== baseline;
  const instructionGroups=groupInstructionFiles(state?.documents,state?.instructionFiles);
 const agentInstructionGroups=instructionGroups.filter(group=>!instructionAgent||group.agents.includes(instructionAgent));
 const currentInstructionGroup=instructionGroups.find(group=>group.path===file?.Path);
 const settingsDirty = plannerBaseline !== null && JSON.stringify(planner) !== JSON.stringify(plannerBaseline);
  async function refresh(reset = false) {
    const fresh = await call("Snapshot");
    setState(fresh);
    if (reset) { setPlanner(fresh.planner); setPlannerBaseline(fresh.planner); }
    if (reset) {
      setText(fresh.shared);
      setBaseline(fresh.shared);
      setExpected(fresh.shared);
      setFile(null);
      setInstructionAgent("");
      setCurrentInstructions("");
      setMode("replace");
      setReview(null);
      setTab("edit");
    }
    return fresh;
  }
  async function perform(action) {
    setError("");
    setBusy(true);
    try {
      return await action();
    } catch (e) {
      setError(String(e.message || e));
      return null;
    } finally {
      setBusy(false);
    }
  }
  useEffect(() => {
    if (initialized.current) return;
    initialized.current = true;
    perform(async () => {
      const fresh = await refresh(true);
      if (!isNative() && new URLSearchParams(location.search).has("review")) {
        setSection("instructions");
        const draft = fresh.shared + "\n" + initialAddition;
        setText(draft);
        const r = await call("ReviewInstructions", {
          text: draft,
          expected: fresh.shared,
        });
        setReview(r);
        setTab("review");
      }
    });
  }, []);
  useEffect(() => () => clearInterval(jobTimer.current), []);
  useEffect(() => {
    if (!job || job.status !== "running") return;
    const timer = setInterval(
      () => setElapsed(Math.floor((Date.now() - job.started) / 1000)),
      1000,
    );
    return () => clearInterval(timer);
  }, [job?.id, job?.status]);
  useEffect(() => {
    if (isNative() && window.go.client.App.SetDirty)
      window.go.client.App.SetDirty(dirty || settingsDirty || Boolean(request.trim()) || Boolean(plan));
  }, [dirty,settingsDirty,request,plan]);
  function navigate(target) {
    if (target === section) return;
    if (dirty || settingsDirty) {
      setDialog({ type: "discard", target });
      return;
    }
    setSection(target);
    setNotice("");
    setError("");
    setQuery("");
  }
  function discard() {
    if (settingsDirty) setPlanner(plannerBaseline);
    setText(baseline);
    setMode("replace");
    setReview(null);
    setTab("edit");
    setNotice("Draft discarded. Saved instructions were not changed.");
  }
  function edit(value) {
    setText(value);
    setReview(null);
    setSavedResults([]);
    setNotice("");
  }
  async function openFile(doc) {
    if (dirty) {
      setDialog({ type: "open-file", doc });
      return;
    }
    await loadFile(doc);
  }
  async function loadFile(doc) {
    return await perform(async () => {
      if (!doc) {
        await refresh(true);
        return true;
      }
      const view = await call("ReadPersonalDocument", doc.Agent, doc.Path);
      setFile(doc);
      setText(view.text);
      setCurrentInstructions(view.current ?? view.expected ?? "");
      setBaseline(view.text);
      setExpected(view.expected);
      setMode("replace");
      setReview(null);
      setTab("current");
      setNotice(
        "Showing saved global instructions for this agent. Edit personal text separately; shared guidance stays protected.",
      );
      return true;
    });
  }
 async function chooseInstructionAgent(agent) {
 if(dirty){setDialog({type:"instruction-agent",agent});return;}
 await loadInstructionAgent(agent);
 }
 async function loadInstructionAgent(agent) {
 if(!agent){if(await loadFile(null)){setInstructionAgent("")}return;}
 const group=preferredInstructionFile(instructionGroups,agent);
 if(!group){setInstructionAgent(agent);setFile(null);setReview(null);setTab("current");setText("");setBaseline("");setExpected("");setCurrentInstructions("");return;}
 const doc=group.documents.find(d=>d.Agent===agent);
 if(await loadFile(doc)){setInstructionAgent(agent)}
 }
 async function reloadInstructionSources(){
 const fresh=await perform(()=>refresh(false));
 if(fresh && file && !dirty){
 const doc=fresh.documents?.find(d=>d.Agent===file.Agent&&d.Path===file.Path);
 if(doc){await loadFile(doc)}else{setFile(null);setInstructionAgent("");setText(fresh.shared);setBaseline(fresh.shared);setExpected(fresh.shared);setTab("edit");setReview(null);setNotice("The selected file is no longer available. Instruction sources were refreshed.");}
 }
 }
  async function reviewText() {
    await perform(async () => {
      const r = await call("ReviewInstructions", {
        agent: file?.Agent || "",
        path: file?.Path || "",
        personal: Boolean(file),
        text,
        expected,
        append: mode === "append",
      });
      setReview(r);
      setTab("review");
      setSavedResults([]);
    });
  }
  async function apply() {
    await perform(async () => {
      const result = await call("ApplyReview", review.token);
      setNotice(result.message);
      setSavedResults(result.results || []);
      const fresh = await refresh(false);
      if (file) {
        const view = await call("ReadPersonalDocument", file.Agent, file.Path);
        setText(view.text);
      setCurrentInstructions(view.current ?? view.expected ?? "");
        setBaseline(view.text);
        setExpected(view.expected);
      } else {
        setText(fresh.shared);
        setBaseline(fresh.shared);
        setExpected(fresh.shared);
        setMode("replace");
      }
      setReview(null);
      setTab("edit");
      if (result.partial) setError(result.message);
    });
  }
  async function reviewAgent(profile, action, server = "") {
 await perform(async () => {
 const r = await call("ReviewAgent", profile, action, server);
 setServerReview(r);setDialog({type:"server",action,name:server});
 });
 }
 async function restoreAgent(profile) {
 await perform(async()=>{const r=await call("ReviewRecovery","restore",profile);setServerReview(r);setDialog({type:"server",action:"restore"});});
 }
 async function retrySync() {
 await perform(async () => { const r=await call("ReviewRecovery","retry","");setServerReview(r);setDialog({type:"server",action:"retry"}); });
 }
 async function reviewServer(server, action) {
    if (agentFilter && action !== "remove") return reviewAgent(agentFilter,action,server.name);
    await perform(async () => {
      const r = await call("ReviewServer", server.name, action);
      setServerReview(r);
      setDialog({ type: "server", action, name: server.name });
    });
  }
  async function applyServer() {
    await perform(async () => {
      const result = await call("ApplyReview", serverReview.token);
      setDialog(null);
      setServerReview(null);
      setNotice(result.message);
      await refresh(false);
      if (result.partial) setError(result.message);
    });
  }
  async function watch(id) {
    clearInterval(jobTimer.current);
    let inFlight = false;
    const poll = async () => {
      if (inFlight) return true;
      inFlight = true;
      try {
        const next = await call("Job", id);
        setJob(next);
        if (next.status !== "running") {
          clearInterval(jobTimer.current);
          if (next.plan) {
            setPlan(next.plan);
            setManual(false);
          }
          if (next.kind === "planning" && next.plan)
            setMessages((old) => [
              ...old,
              { speaker: "agent", text: next.plan.summary },
            ]);
          if (next.status === "failed") {
            setError(next.message);
            setMessages((old) => [
              ...old,
              { speaker: "system", text: next.message },
            ]);
          }
          if (next.kind === "installation") {
            setPrivateValues({});
            setMessages((old) => [
              ...old,
              { speaker: "system", text: next.message },
            ]);
            await refresh(false);
            if (next.status === "done") {
              setPlan(null);
              setShowPlan(false);
            }
          }
        }
        return next.status === "running";
      } catch (e) {
        clearInterval(jobTimer.current);
        setError(e.message);
        setJob((old) => ({ ...old, status: "failed" }));
        return false;
      } finally {
        inFlight = false;
      }
    };
    if (await poll()) jobTimer.current = setInterval(poll, 800);
  }
  async function send() {
    if (!request.trim() || job?.status === "running") return;
    await perform(async () => {
      const value = request.trim();
      const id = await call("Plan", value, plan?.id || "");
      setMessages((old) => [...old, { speaker: "you", text: value }]);
      setRequest("");
      setPrivateValues({});
      setShowPlan(false);
      await watch(id);
    });
  }
  async function install() {
    await perform(async () => {
      const id = await call("Install", plan.id, manual, privateValues);
      setPrivateValues({});
      setShowPlan(false);
      await watch(id);
    });
  }
  async function prerequisites(packages, docker) {
    await perform(async () => {
      setDialog(null);
      const id = await call("RepairPrerequisites", plan.id, packages, docker);
      await watch(id);
    });
  }
  const diff = review ? lineDiff(review.current, review.proposed) : null;
  const allowedTargets = review ? automatic(review.results || []) : [];
  const isRunning = job?.status === "running";
 const visibleServers=(state?.servers || []).filter(s=>s.name.toLowerCase().includes(query.toLowerCase()));
 const failedAgents=(state?.agents || []).filter(a=>a.syncStatus === "failed");
  const title = file
    ? `Review ${friendly(file.Agent)} instructions`
    : "Review shared instructions";
  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">MCPDeck</div>
        {(!isNative() || state?.demo) && <div className="sample-label">Sample workspace</div>}
        <nav aria-label="Main navigation">
          {navigation.map(([id, label, Icon]) => (
            <button
              key={id}
              className={`nav-item ${section === id ? "active" : ""}`}
              aria-current={section === id ? "page" : undefined}
              onClick={() => navigate(id)}
            >
              <Icon size={25} weight="regular" />
              <span>{label}</span>
            </button>
          ))}
        </nav>
        <button
          className={`nav-item settings ${section === "settings" ? "active" : ""}`}
          onClick={() => navigate("settings")}
        >
          <GearSix size={25} />
          <span>Settings</span>
        </button>
      </aside>
      <main className="main-area" aria-busy={busy}>
 {!state && <div className="notice" role="status">{busy ? "Loading your workspace…" : "Workspace could not be loaded."}{!busy && <Button onClick={()=>perform(()=>refresh(true))}>Retry loading</Button>}</div>}
        {section === "instructions" && (
          <>
            <header className="page-header">
              <h1>
                {tab === "review" && review
                  ? file
                    ? `Review ${friendly(file.Agent)} instructions`
                    : review.title
                  : file
                    ? `${friendly(file.Agent)} instructions`
                    : "Shared instructions"}
              </h1>
              <p>
                {tab === "review" && review
                  ? review.detail
                  : file
                    ? tab === "current" ? "Saved global guidance for the selected file and its agent aliases." : "Edit this file’s personal guidance. Shared guidance and rule settings are preserved."
                    : "Replace only the MCPDeck section. Keep personal and project rules."}
              </p>
            </header>
            {state?.demo && <p className="quiet-note">Sample workspace: these are example instruction files. Open the installed desktop app to see your own agents.</p>}
            <div className="instruction-browser">
 <label>Agent<select aria-label="Instruction agent" disabled={busy} value={instructionAgent} onChange={e=>chooseInstructionAgent(e.target.value)}><option value="">All agents / shared guidance</option>{state?.agents?.map(agent=><option key={agent.name} value={agent.name}>{friendly(agent.name)}</option>)}</select></label>
 <label>Instruction file<select id="instruction-file" aria-label="Instruction file" disabled={busy} value={file?.Path || "shared"} onChange={e=>{if(e.target.value==="shared"){chooseInstructionAgent("");return;}const group=instructionGroups.find(g=>g.path===e.target.value);openFile(group?.documents.find(d=>d.Agent===instructionAgent) || group?.documents[0]);}}><option value="shared">Shared guidance · all agents</option>{agentInstructionGroups.map(group=><option key={group.path} value={group.path}>{group.agents.map(friendly).join(" / ")} · {group.path.split(/[\\/]/).pop()} · {group.status}</option>)}</select></label>
 <Button disabled={busy} onClick={reloadInstructionSources}><ArrowClockwise size={18}/>Reload sources</Button>
 </div>
 {file && <p className="quiet-note">File used by: {currentInstructionGroup?.agents.map(friendly).join(" / ")}. {currentInstructionGroup?.status === "not created" ? "This global file has not been created yet. It can be created through personal editing and review." : currentInstructionGroup?.status === "empty" ? "The saved file is empty." : "Editing this file affects every agent listed here."}</p>}
            <div className="tabs">
 {file && <button className={tab === "current" ? "selected" : ""} onClick={()=>setTab("current")}>Current instructions</button>}
              <button
                className={tab === "edit" ? "selected" : ""}
                onClick={() => setTab("edit")}
              >
                {file ? "Edit personal text" : "Edit"}
              </button>
              <button
                className={tab === "review" ? "selected" : ""}
                disabled={!review || busy}
                onClick={() => setTab("review")}
              >
                Review
              </button>

            </div>
            <ErrorText text={error} />
            {notice && (
              <div className="notice" role="status">
                <CheckCircle size={20} />
                <span>{notice}</span>
              </div>
            )}
            {instructionAgent && agentInstructionGroups.length === 0 ? (
 <div className="empty-state"><h2>No automatic global instruction file for {friendly(instructionAgent)}</h2><p>{state?.targets?.find(t=>t.Agent===instructionAgent)?.Detail || "Use this agent’s own instruction settings. Project files are not scanned here."}</p><Button onClick={()=>chooseInstructionAgent("")}>Open shared guidance</Button></div>
 ) : tab === "review" && review ? (
              <>
                <div className="comparison">
                  <DiffPane title="Current" items={diff.current} />
                  <DiffPane title="Proposed" items={diff.proposed} added />
                </div>
                <section className="distribution">
                  <h2>Distribution</h2>
                  <Results rows={review.results} />
                </section>
                <footer className="action-footer">
                  <p>
                    {file
                      ? "Shared guidance and rule settings are preserved."
                      : "Other personal text is preserved."}{" "}
                    Agent sessions may need reloading.
                  </p>
                  <div>
                    <Button disabled={busy} onClick={() => setTab("edit")}>
                      Back to editing
                    </Button>
                    <Button
                      primary
                      disabled={busy || allowedTargets.length === 0}
                      onClick={apply}
                    >
                      {busy
                        ? "Applying…"
                        : file
                          ? "Save this file"
                          : `Apply to ${allowedTargets.length} agents`}
                    </Button>
                  </div>
                </footer>
              </>
            ) : tab === "current" && file ? (
 <>
 <div className="editor-toolbar"><span className="file-path" title={file.Path}>{file.Path}</span><Button disabled={busy} onClick={()=>openFile(file)}>Reload saved file</Button></div>
 <textarea className="instruction-editor" aria-label="Current saved agent instructions" value={currentInstructions} readOnly spellCheck={false} placeholder="This global instruction file has no saved guidance yet." />
 <p className="quiet-note">Saved contents of this file, including its MCPDeck shared guidance. Project rules and other files may add instructions in an agent session.</p>
 <footer className="action-footer"><p>{dirty ? "Your personal draft has unsaved changes." : "Read-only view of the saved file"}</p><Button onClick={()=>setTab("edit")}>Edit personal text</Button></footer>
 </>
            ) : (
              <>
                <div className="editor-toolbar">
                  <div>
                    {!file && (
                      <label>
                        Action{" "}
                        <select
                          aria-label="Shared instruction action"
                          value={mode}
                          onChange={(e) => {
                            if (dirty) {
                              setDialog({ type: "mode", mode: e.target.value });
                              return;
                            }
                            setMode(e.target.value);
                            setText(
                              e.target.value === "append" ? "" : baseline,
                            );
                            setReview(null);
                          }}
                        >
                          <option value="replace">
                            Replace shared section
                          </option>
                          <option value="append">Append an instruction</option>
                        </select>
                      </label>
                    )}
                    {file && (
                      <span title={file.Path} className="file-path">
                        {file.Path}
                      </span>
                    )}
                  </div>
                  <div>
                    <button
                      className="text-button"
                      disabled={busy || !text}
                      onClick={() => setDialog({ type: "clear" })}
                    >
                      Clear text
                    </button>
                    {file && (
                      <button
                        className="text-button"
                        onClick={() => setDialog({ type: "use-shared" })}
                      >
                        Use for all agents
                      </button>
                    )}
                    <button
                      className="icon-button"
                      title="Undo typing (use Cmd+Z / Ctrl+Z)"
                      aria-label="Focus editor to undo typing"
                      onClick={() => {
                        editRef.current?.focus();
                        document.execCommand("undo");
                      }}
                    >
                      <ArrowCounterClockwise size={21} />
                    </button>
                  </div>
                </div>
                <textarea
                  ref={editRef}
                  className="instruction-editor"
                  aria-label={
                    file ? "Personal instructions" : "Shared instructions"
                  }
                  value={text}
                  onChange={(e) => edit(e.target.value)}
                  spellCheck={false}
                  placeholder={
                    mode === "append"
                      ? "Write an instruction to append to shared guidance."
                      : "Write your shared working preferences…"
                  }
                  disabled={busy}
                  onKeyDown={(e) => {
                    if (
                      (e.metaKey || e.ctrlKey) &&
                      e.key.toLowerCase() === "s"
                    ) {
                      e.preventDefault();
                      if (!busy) reviewText();
                    }
                  }}
                />
                <div className="editor-meta">
                  <span>
                    {new TextEncoder().encode(text).length.toLocaleString()}{" "}
                    bytes ·{" "}
                    {file
                      ? "Personal text only; shared block protected"
                      : "Global guidance; project rules excluded"}
                  </span>
                  <span>Cmd / Ctrl+A selects all</span>
                </div>
                {savedResults.length > 0 && (
                  <section className="saved-results">
                    <h2>Latest distribution</h2>
                    <Results rows={savedResults} />
                  </section>
                )}
                <footer className="action-footer">
                  <p>
                    {dirty
                      ? "Unsaved changes"
                      : mode === "append"
                        ? "Append to the end of shared guidance"
                        : file
                          ? "This file only"
                          : "Shared guidance is up to date"}
                  </p>
                  <div>
                    <Button disabled={busy || !dirty} onClick={discard}>
                      Discard
                    </Button>
                    <Button
                      primary
                      disabled={busy || !state || (!dirty && mode !== "append")}
                      onClick={reviewText}
                    >
                      {busy ? "Preparing review…" : "Review changes"}
                    </Button>
                  </div>
                </footer>
              </>
            )}
          </>
        )}
        {section === "servers" && (
          <>
            <header className="page-header header-with-action">
              <div>
                <h1>MCP servers</h1>
                <p>
                  Managed connections and MCPs found in your agent settings.
                </p>
              </div>
              <Button primary onClick={() => navigate("new")}>
                <Plus size={20} />
                New MCP
              </Button>
            </header>
            <ErrorText text={error} />
            {notice && (
              <div className="notice" role="status">
                {notice}
              </div>
            )}
            <div className="list-toolbar">
 <label className="agent-filter">Agent <select aria-label="Server target agent" value={agentFilter} onChange={e=>setAgentFilter(e.target.value)}><option value="">All configured agents</option>{state?.agents?.map(a=><option key={a.name} value={a.name}>{friendly(a.name)}</option>)}</select></label>
              <label className="search">
                <MagnifyingGlass size={20} />
                <input
                  aria-label="Search MCP servers"
                  placeholder="Search servers"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                />
              </label>
              <Button
                onClick={() => perform(() => refresh(false))}
                disabled={busy}
              >
                <ArrowClockwise size={18} />
                Refresh
              </Button>
            </div>
            {state?.warnings?.map((w) => (
              <ErrorText key={w} text={w} />
            ))}
            <div className="table-scroll">
              <table className="servers-table">
                <thead>
                  <tr>
                    <th>Server</th>
                    <th>Source</th>
                    <th>Agent settings</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {visibleServers.map((s) => (
                      <tr key={s.name}>
                        <td>
                          <strong>{s.name}</strong>
                          <small>{s.kind} · Connection not tested</small>
                        </td>
                        <td>
                          {s.managed ? "MCPDeck" : "Discovered in agents"}
                        </td>
                        <td>
                          {s.enabled
                            ? s.profiles.map(friendly).join(", ")
                            : "Disabled in all targets"}
                        </td>
                        <td>
                          <div className="row-actions">
                            {s.managed && (
                              <Button
                                disabled={busy || isRunning}
                                onClick={() =>
                                  reviewServer(
                                    s,
                                    (agentFilter ? s.profiles.includes(agentFilter) : s.enabled) ? "disable" : "enable",
                                  )
                                }
                              >
                                {(agentFilter ? s.profiles.includes(agentFilter) : s.enabled) ? "Disable" : "Enable"}
                              </Button>
                            )}
                            <button
                              className="text-button destructive"
                              disabled={busy || isRunning}
                              onClick={() => reviewServer(s, "remove")}
                            >
                              Remove everywhere
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                </tbody>
              </table>
            </div>
            {state?.servers?.length > 0 && visibleServers.length === 0 && <p className="empty-state">No servers match this search.</p>}
            {state?.servers?.length === 0 && (
              <div className="empty-state">
                <h2>No MCP servers yet</h2>
                <p>
                  Add an MCP by name. The connected agent will prepare a
                  reviewed installation recipe.
                </p>
                <Button primary onClick={() => navigate("new")}>
                  Add your first MCP
                </Button>
              </div>
            )}
          </>
        )}
        {section === "agents" && (
          <>
            <header className="page-header header-with-action">
              <div>
                <h1>Agents</h1>
                <p>
                  Configured integration paths. A detected agent is not proof of
                  a live MCP connection.
                </p>
              </div>
              <Button
                disabled={busy}
                onClick={() => perform(() => refresh(false))}
              >
                <ArrowClockwise size={18} />
                Refresh
              </Button>
            </header>
            <ErrorText text={error} />
 {notice && <div className="notice" role="status">{notice}</div>}
 {failedAgents.length>0 && <div className="notice"><span>{failedAgents.length} agent synchronization(s) failed. Saved selections are kept.</span><Button disabled={busy || isRunning} onClick={retrySync}>Review retry</Button></div>}
 {state?.warnings?.map(w=><ErrorText key={w} text={w}/>)}
            <div className="table-scroll">
              <table className="agents-table">
                <thead>
                  <tr>
                    <th>Agent</th>
                    <th>Detection</th>
                    <th>Enabled MCPs</th>
                    <th>Mode / Last sync</th>
                    <th>Configuration</th>
 <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {state?.agents?.map((a) => (
                    <tr key={a.name}>
                      <td>{friendly(a.name)}</td>
                      <td>{a.detected ? "Detected" : "Not detected"}</td>
                      <td>{a.enabled}</td>
 <td>{a.mode}<small>{a.syncStatus || "Not synced yet"}</small></td>
                      <td>
                        <span className="file-path" title={a.path}>{a.path}</span>
                      </td>
 <td><div className="row-actions"><Button disabled={busy} onClick={()=>{setSection("instructions");chooseInstructionAgent(a.name)}}>Instructions</Button><Button disabled={busy || isRunning} onClick={()=>reviewAgent(a.name,"sync")}>Sync</Button><Button disabled={busy || isRunning} onClick={()=>reviewAgent(a.name,a.mode === "bridge" ? "direct" : "bridge")}>Use {a.mode === "bridge" ? "Direct" : "Bridge"}</Button><Button disabled={busy || isRunning} onClick={()=>restoreAgent(a.name)}>Restore backup</Button></div></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <p className="quiet-note">
              Shared files can serve multiple agents. Unsupported global
              instruction targets are shown as manual during review.
            </p>
          </>
        )}
        {section === "new" && (
          <>
            <header className="page-header">
              <h1>Connect an MCP</h1>
              <p>
                Describe what you need. Review the agent’s recipe before any
                program runs.
              </p>
            </header>
            <div className="conversation-toolbar">
              <span>
                Connected planner:{" "}
                <strong>{planner.provider || "codex"}</strong>
              </span>
              <button
                className="text-button"
                onClick={() => navigate("settings")}
              >
                Change planner
              </button>
              {plan && (
                <Button onClick={() => setShowPlan(!showPlan)}>
                  {showPlan ? "Back to conversation" : "Review installation"}
                </Button>
              )}
            </div>
            <ErrorText text={error} />
            {showPlan && plan ? (
              <section className="plan-review">
                <h2>{plan.name}</h2>
                <p>{plan.summary}</p>
                <h3>Commands to approve</h3>
                <ol>
                  {plan.steps?.map((step, i) => (
                    <li key={i}>
                      <strong>{step.description}</strong>
                      <pre>
                        {step.command} {JSON.stringify(step.args)}
                        <br />
                        Directory: {step.directory || "."}
                      </pre>
                    </li>
                  ))}
                </ol>
                <h3>Agent targets</h3>
                <p>
                  {plan.targets?.map(friendly).join(", ") ||
                    "No targets detected"}
                </p>
                <h3>Documentation</h3>
                {plan.sources?.map((source) => (
                  <a
                    key={source}
                    href={/^https:\/\//.test(source) ? source : undefined}
                    target="_blank"
                    rel="noreferrer"
                  >
                    {source}
                  </a>
                ))}
                {plan.prerequisiteError && (
                  <div className="prerequisites">
                    <ErrorText text={plan.prerequisiteError} />
                    {plan.missingPackages?.length > 0 && (
                      <Button
                        onClick={() =>
                          setDialog({ type: "prerequisites", packages: true })
                        }
                      >
                        Review prerequisite install
                      </Button>
                    )}
                    {/docker/i.test(plan.prerequisiteError) && (
                      <Button
                        onClick={() =>
                          setDialog({ type: "prerequisites", docker: true })
                        }
                      >
                        Review starting Colima
                      </Button>
                    )}
                    <p>
                      Ask the planner for help if a prerequisite cannot be
                      completed automatically.
                    </p>
                  </div>
                )}
                {plan.manual?.length > 0 && (
                  <section>
                    <h3>Manual prerequisites</h3>
                    <ul>
                      {plan.manual.map((item, i) => (
                        <li key={i}>{item}</li>
                      ))}
                    </ul>
                    <label className="check">
                      <input
                        type="checkbox"
                        checked={manual}
                        onChange={(e) => setManual(e.target.checked)}
                      />
                      I completed or independently verified these prerequisites.
                    </label>
                  </section>
                )}
                {plan.required?.length > 0 && (
                  <section className="private-inputs">
                    <h3>Private connection values</h3>
                    <p>
                      Kept out of chat and planner requests. Saved in the local
                      protected MCPDeck configuration.
                    </p>
                    {plan.required.map((key) => (
                      <label key={key}>
                        {key}
                        <input
                          type="password"
                          autoComplete="off"
                          value={privateValues[key] || ""}
                          onChange={(e) =>
                            setPrivateValues((old) => ({
                              ...old,
                              [key]: e.target.value,
                            }))
                          }
                        />
                      </label>
                    ))}
                  </section>
                )}
                {plan.placeholder && (
                  <ErrorText text="This recipe still contains a placeholder. Ask the agent to complete it before installing." />
                )}
                <footer className="action-footer">
                  <p>
                    These exact commands can download and run programs with your
                    user permissions.
                  </p>
                  <div>
                    <Button onClick={() => setShowPlan(false)}>
                      Ask a question
                    </Button>
                    <Button
                      primary
                      disabled={
                        busy ||
                        isRunning ||
                        plan.placeholder ||
                        Boolean(plan.prerequisiteError) ||
                        !plan.targets?.length ||
                        (plan.manual?.length > 0 && !manual) ||
                        (plan.required || []).some((key) => !privateValues[key])
                      }
                      onClick={() => setDialog({ type: "install" })}
                    >
                      Approve installation
                    </Button>
                  </div>
                </footer>
              </section>
            ) : (
              <>
                <div className="conversation" aria-live="polite">
                  {messages.length === 0 ? (
                    <div className="conversation-empty">
                      <h2>What would you like to connect?</h2>
                      <p>
                        Use an MCP name and describe its purpose. The agent
                        checks available software and prepares the installation
                        steps.
                      </p>
                      <div className="suggestions">
                        {[
                          "Connect Oracle for read-only database queries.",
                          "Add the official Git MCP for a local repository.",
                          "Connect Sentry to inspect issues.",
                        ].map((value) => (
                          <button
                            key={value}
                            onClick={() => {
                              setRequest(value);
                              document.getElementById("mcp-request")?.focus();
                            }}
                          >
                            {value}
                            <ArrowRight size={18} />
                          </button>
                        ))}
                      </div>
                    </div>
                  ) : (
                    messages.map((m, i) => (
                      <article className={`message ${m.speaker}`} key={i}>
                        <h3>
                          {m.speaker === "you"
                            ? "You"
                            : m.speaker === "agent"
                              ? planner.provider
                              : "MCPDeck"}
                        </h3>
                        <p>{m.text}</p>
                      </article>
                    ))
                  )}
                  {job && (
                    <div className="job-status">
                      <div>
                        <strong>
                          {job.status === "running"
                            ? "Working"
                            : job.status === "failed"
                              ? "Needs attention"
                              : "Finished"}
                        </strong>
                        {job.status === "running" && (
                          <span>{elapsed}s elapsed</span>
                        )}
                      </div>
                      <p>{job.message}</p>
                      {job.events?.length > 0 && (
                        <details>
                          <summary>Progress details</summary>
                          {job.events.map((event, i) => (
                            <p key={i}>{event}</p>
                          ))}
                        </details>
                      )}
                      {isRunning && (
                        <Button onClick={() => perform(() => call("Cancel"))}>
                          <Stop size={16} />
                          Stop operation
                        </Button>
                      )}
                    </div>
                  )}
                  {plan && !isRunning && (
                    <Button primary onClick={() => setShowPlan(true)}>
                      Review installation
                    </Button>
                  )}
                </div>
                <form
                  className="composer"
                  onSubmit={(e) => {
                    e.preventDefault();
                    send();
                  }}
                >
                  <label htmlFor="mcp-request">
                    {plan ? "Ask a follow-up question" : "MCP name or request"}
                  </label>
                  <textarea
                    id="mcp-request"
                    value={request}
                    onChange={(e) => setRequest(e.target.value)}
                    placeholder="For example: Connect Oracle using my existing SQLcl installation."
                    disabled={isRunning || busy}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" && !e.shiftKey) {
                        e.preventDefault();
                        send();
                      }
                    }}
                  />
                  <div>
                    <p>
                      Account usage may apply. Keep passwords and tokens out of
                      chat.
                    </p>
                    <Button
                      primary
                      type="submit"
                      disabled={!request.trim() || isRunning || busy}
                    >
                      <PaperPlaneRight size={19} />
                      {plan ? "Ask agent" : "Prepare installation"}
                    </Button>
                  </div>
                </form>
              </>
            )}
          </>
        )}
        {section === "settings" && (
          <>
            <header className="page-header">
              <h1>Settings</h1>
              <p>
                Choose the planning provider used for new MCP installations.
              </p>
            </header>
            <ErrorText text={error} />
            {notice && (
              <div className="notice" role="status">
                {notice}
              </div>
            )}
            <form
              className="settings-form"
              onSubmit={(e) => {
                e.preventDefault();
                perform(async () => {
                  await call("SavePlanner", planner);
                  setPlannerBaseline({...planner});
                  setNotice("Planner settings saved.");
                  await refresh(false);
                });
              }}
            >
              <label>
                Planning provider
                <select
                  value={planner.provider || "codex"}
                  onChange={(e) => setPlanner({ provider: e.target.value })}
                >
                  {[
                    "codex",
                    "claude",
                    "gemini",
                    "openai-api",
                    "anthropic-api",
                    "gemini-api",
                    "openai-compatible",
                  ].map((value) => (
                    <option key={value}>{value}</option>
                  ))}
                </select>
              </label>
              <p>
                CLI providers use your existing signed-in installation. API
                credentials are referenced by environment variable, never
                entered into chat.
              </p>
              <label>
                Model{" "}
                {planner.provider?.includes("api")
                  ? "(required)"
                  : "(optional)"}
                <input
                  value={planner.model || ""}
                  onChange={(e) =>
                    setPlanner((old) => ({ ...old, model: e.target.value }))
                  }
                />
              </label>
              {(planner.provider?.includes("api") ||
                planner.provider === "openai-compatible") && (
                <label>
                  API key environment variable
                  <input
                    autoComplete="off"
                    placeholder="For example: OPENAI_API_KEY"
                    value={planner.key_env || ""}
                    onChange={(e) =>
                      setPlanner((old) => ({ ...old, key_env: e.target.value }))
                    }
                  />
                </label>
              )}
              {planner.provider === "openai-compatible" && (
                <label>
                  Compatible API base URL
                  <input
                    value={planner.base_url || ""}
                    onChange={(e) =>
                      setPlanner((old) => ({
                        ...old,
                        base_url: e.target.value,
                      }))
                    }
                  />
                </label>
              )}
              <Button primary type="submit" disabled={busy || isRunning || !settingsDirty}>
                Save planner
              </Button>
              <section className="config-note">
                <h2>Local configuration</h2>
                <code>{state?.config}</code>
                <p>
                  The terminal and desktop client share this configuration.
                  Project instruction files are excluded.
                </p>
                {(!isNative() || state?.demo) && (
                  <p>
                    This browser preview uses temporary sample files. It does
                    not change your real agent settings or call provider
                    accounts.
                  </p>
                )}
              </section>
            </form>
          </>
        )}
      </main>
      {dialog && (
        <Dialog
          busy={busy}
          title={
            {
              clear: "Clear this draft?",
              discard: "Discard unsaved changes?",
              mode: "Change editing action?",
              "open-file": "Open another file?",
 "instruction-agent":"Switch instruction agent?",
              "use-shared": "Use this guidance for all agents?",
              server: serverReview?.title,
              install: "Run this reviewed installation?",
              prerequisites: "Approve prerequisite action?",
            }[dialog.type]
          }
          onClose={() => {if (!busy) setDialog(null)}}
          footer={(() => {
            const cancel = (
              <Button onClick={() => setDialog(null)} disabled={busy}>
                Keep editing
              </Button>
            );
            if (dialog.type === "server")
              return (
                <>
                  {cancel}
                  <Button
                    danger={["remove","restore"].includes(dialog.action)}
                    primary={!["remove","restore"].includes(dialog.action)}
                    disabled={busy}
                    onClick={applyServer}
                  >
                    Confirm {dialog.action}
                  </Button>
                </>
              );
            if (dialog.type === "install")
              return (
                <>
                  {cancel}
                  <Button
                    primary
                    disabled={busy}
                    onClick={() => {
                      setDialog(null);
                      install();
                    }}
                  >
                    Run reviewed commands
                  </Button>
                </>
              );
            if (dialog.type === "prerequisites")
              return (
                <>
                  {cancel}
                  <Button
                    primary
                    disabled={busy}
                    onClick={() =>
                      prerequisites(
                        Boolean(dialog.packages),
                        Boolean(dialog.docker),
                      )
                    }
                  >
                    Approve and continue
                  </Button>
                </>
              );
            return (
              <>
                {cancel}
                <Button
                  primary
                  onClick={() => {
                    const d = dialog;
                    setDialog(null);
                    if (d.type === "clear") {
                      edit("");
                      editRef.current?.focus();
                    }
                    if (d.type === "discard") {
                      discard();
                      setSection(d.target);
                    }
                    if (d.type === "mode") {
                      setMode(d.mode);
                      setText(d.mode === "append" ? "" : baseline);
                      setReview(null);
                    }
                    if(d.type === "instruction-agent"){discard();loadInstructionAgent(d.agent);}
                    if (d.type === "open-file") {
                      discard();
                      loadFile(d.doc);
                    }
                    if (d.type === "use-shared") {
                      const personal = text;
                      perform(async () => {
                        const fresh = await refresh(true);
                        setText(personal);
                        setBaseline(fresh.shared);
                        setNotice(
                          "Loaded a shared draft. Review before distributing to all supported agents.",
                        );
                      });
                    }
                  }}
                >
                  {dialog.type === "clear"
                    ? "Clear draft"
                    : dialog.type === "use-shared"
                      ? "Prepare shared draft"
                      : "Discard and continue"}
                </Button>
              </>
            );
          })()}
        >
          <div className="dialog-body">
 <ErrorText text={error} />
            {dialog.type === "server" ? (
              <>
                <p>{serverReview?.detail}</p>
                <Results rows={serverReview?.results || []} />
              </>
            ) : dialog.type === "install" ? (
              <>
                <p>
                  The reviewed commands will run with your user permissions.
                  Private values are supplied only to the MCP connection.
                </p>
                <p>
                  Approval is bound to this exact recipe. Changing the plan
                  requires a new review.
                </p>
              </>
            ) : dialog.type === "prerequisites" ? (
              <>
                <p>
                  {dialog.packages
                    ? "Install these missing packages with Homebrew:"
                    : "Start the existing Colima Docker runtime:"}
                </p>
                <code>
                  {dialog.packages
                    ? `brew install ${(plan?.missingPackages || []).join(" ")}`
                    : "colima start"}
                </code>
              </>
            ) : (
              <p>
                {dialog.type === "clear"
                  ? file
                    ? "Clear personal draft text. The shared MCPDeck section and rule settings are kept. Nothing is saved until review and approval."
                    : "Clear this draft. Saving an empty shared draft removes only MCPDeck sections; other personal text stays."
                  : dialog.type === "use-shared"
                    ? "Prepare this personal guidance as a replacement shared draft. No file is changed until you review and apply it."
                    : "The current draft has not been saved. You can keep editing or discard it."}
              </p>
            )}
          </div>
        </Dialog>
      )}
    </div>
  );
}
