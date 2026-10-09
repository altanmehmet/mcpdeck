export const isNative = () => Boolean(window.go?.client?.App);
export async function call(method, ...args) {
  const native = window.go?.client?.App;
  if (native) {
 if (typeof native[method] !== "function") throw new Error("This desktop version does not support this action. Restart the updated app.");
 return native[method](...args);
 }
  const response = await fetch("/api/client", {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-MCPDeck-Preview": "1" },
    body: JSON.stringify({ method, args }),
  });
  if (!response.ok)
    throw new Error(
      "The local client connection is unavailable. Restart the preview or native app.",
    );
  const result = await response.json();
  if (result.error) throw new Error(result.error);
  return result.value;
}
export function friendly(name) {
  const labels = {
      codex: "Codex",
      "claude-code": "Claude Code",
      claude: "Claude Desktop",
      "copilot-cli": "Copilot CLI",
      copilot: "Copilot in VS Code",
      cursor: "Cursor",
      "gemini-cli": "Gemini CLI",
      antigravity: "Antigravity",
      "qwen-code": "Qwen Code",
      opencode: "OpenCode",
      windsurf: "Windsurf",
      kiro: "Kiro",
};
 return Object.prototype.hasOwnProperty.call(labels,name) ? labels[name] : name;
}
