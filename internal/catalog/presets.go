package catalog

import "github.com/altanmehmet/mcpdeck/internal/model"

// Presets are opt-in. Legacy reference servers may require replacement before production use.
func Presets() map[string]model.ServerConfig {
	return map[string]model.ServerConfig{
		"filesystem":   {Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-filesystem", "${MCP_FILESYSTEM_ROOT}"}},
		"github":       {Command: "docker", Args: []string{"run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN", "ghcr.io/github/github-mcp-server"}, Env: map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_PERSONAL_ACCESS_TOKEN}"}},
		"postgres":     {Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-postgres", "${POSTGRES_URL}"}},
		"sqlite":       {Command: "uvx", Args: []string{"mcp-server-sqlite", "--db-path", "${SQLITE_DB_PATH}"}},
		"brave-search": {Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-brave-search"}, Env: map[string]string{"BRAVE_API_KEY": "${BRAVE_API_KEY}"}},
		"fetch":        {Command: "uvx", Args: []string{"mcp-server-fetch"}},
	}
}
