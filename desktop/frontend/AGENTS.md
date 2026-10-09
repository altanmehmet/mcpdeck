# Prototype Instructions

Run the local server yourself and open the preview in the browser available to this environment. Do not give the user server-start instructions when you can run it.

Before making substantial visual changes, use the Product Design plugin's `get-context` skill when the visual source is unclear or no longer matches the current goal. When the user gives durable prototype-specific design feedback, preferences, or decisions, record them in `AGENTS.md`.

When implementing from a selected generated mock, treat that image as the source of truth for layout, component anatomy, density, spacing, color, typography, visible content, and hierarchy.

Build app UI in `src/`. Keep `.openai/hosting.json`, `worker/index.js`, `scripts/prepare-sites-build.mjs`, and `tests/sites-worker.test.mjs` intact so the same local prototype can be handed to Sites. Before a Sites handoff, run `npm run build` and `npm run test:sites`; the build must leave `dist/client/index.html`, `dist/server/index.js`, and `dist/.openai/hosting.json`.

## Durable product requirements

- Keep the desktop distributions light and easy to install on macOS and Windows.
- Use the system WebView; do not bundle Chromium, Node, package-manager caches or developer dependencies.
- Agent instruction browsing must show the selected agent's currently saved guidance, including the shared block; keep personal editing separate so shared guidance is protected.
- Clearly distinguish the isolated sample workspace from real agent files.

- Instruction sources must distinguish missing/empty files from read errors and offer explicit agent selection.
- Group a shared physical file once and show every agent affected by edits; never merge separate custom directories by brand alone.

- The instruction manager covers only general user/system custom instructions. Project and repository instructions are excluded, including PACMAN rules.
