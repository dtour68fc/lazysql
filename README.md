# lazysql

A minimal, terminal-first database client for people who live in Vim and tmux. `lazysql` keeps its UI out of your way and your hands on the home row — a focused explorer, a real Vim editor for your queries, and a results viewer, side by side.

It currently supports **PostgreSQL**.

![Main UI — explorer, editor, viewer](assets/main-ui.png)

---

## Why lazysql

- **Clean, uncluttered TUI.** Three panes, one footer hint line, no chrome. Nothing on the screen you didn't ask for.
- **Vim bindings everywhere.** `h j k l` to move, `?` for help, `m` to toggle, `Shift+Tab` to switch panes. The query editor is a full Vim buffer powered by [`vimtea`](https://github.com/kujtimiihoxha/vimtea) — modes, motions, visual selection, all of it.
- **Three connection modes — including a *command* mode.** Plug in static credentials, paste a connection URL, *or* point `lazysql` at a shell command that prints credentials on stdout. The command runs every time you connect, which makes rotating secrets (Vault, AWS RDS IAM auth, GCP IAM, short-lived dev tokens) painless.
- **Session logs for your AI agent.** Every session writes a log file you can hand to Claude / Gemini / your agent of choice to give it concrete database context — the queries you ran, the schemas you touched — without copy-pasting.

---

## Screenshots

### Connection Manager — *command* mode

A connection that fetches its credentials by shell-executing `~/test_creds.sh` on every connect.

![Connection Manager](assets/connection-manager.png)

### Main UI

Left pane: the explorer (databases → schemas → tables → data / schema / indexes).
Top-right: the Vim editor.
Bottom-right: query results, scrollable in both axes.

![Main UI](assets/main-ui.png)

---

## Install

```bash
git clone https://github.com/umairabid/lazysql
cd lazysql
go build -o lazysql .
./lazysql
```

Requires Go 1.25+.

---

## Connection modes

Open a connection in edit mode (`e`) and press `m` to cycle: **credentials → command → url → credentials**.

### 1. Credentials

The familiar form: driver, name, host, port, user, password.

### 2. URL

A single PostgreSQL connection string:

```
postgres://user:password@host:5432/dbname?sslmode=require
```

### 3. Command — *for rotating credentials*

Provide a path to a script. `lazysql` executes it on connect; whatever the script prints is parsed as the credentials for that session. Pair this with anything that hands you short-lived secrets:

```bash
# ~/test_creds.sh — example for Vault
vault kv get -format=json secret/db/prod | jq -r '.data.data | "\(.host)\t\(.user)\t\(.password)\t\(.port)"'
```

You never store a password on disk, and you never re-edit the connection when secrets rotate — `lazysql` just reruns the script.

---

## Session logs (give your AI agent real context)

Each running `lazysql` process writes to:

```
~/.config/lazysql/sessions/session-<pid>.log
```

The log accumulates the activity of the session — useful as a context drop for an LLM agent that's helping you write queries, debug schemas, or summarize a debugging session. Workflow:

```bash
# In another pane, while lazysql is running:
cat ~/.config/lazysql/sessions/session-*.log | claude "summarize what I was doing"
```

Stale logs from dead processes are cleaned up on the next startup. Logs from a *normally exited* session are removed on quit — copy them out while the session is live if you want to keep them.

---

## Keybindings

### Global

| Binding | Action |
| :--- | :--- |
| `Ctrl+c` | Quit |
| `Shift+Tab` | Cycle focus between Explorer → Editor → Viewer |

### Connection Manager

| Binding | Action |
| :--- | :--- |
| `j` / `k` (or `↓` / `↑`) | Move selection |
| `Enter` | Connect to the selected entry |
| `e` | Edit the selected connection |
| `s` | Save the current connection |
| `m` | Cycle connection mode (credentials → command → url) |
| `?` | Toggle help dialog |
| `Esc` | Cancel edit / close help |

In edit mode:

| Binding | Action |
| :--- | :--- |
| `Tab` / `Shift+Tab` | Move between form fields |

### Explorer (left pane)

| Binding | Action |
| :--- | :--- |
| `j` / `k` | Move down / up |
| `l` | Expand node — open a database, list tables, load a table's data / schema / indexes |
| `h` | Collapse / move to parent |

### Editor (top-right pane)

A full Vim buffer via `vimtea` — all your usual normal/insert/visual motions work.

| Binding | Action |
| :--- | :--- |
| `Ctrl+r` | Run query. In *visual* mode, runs the selection; otherwise runs the whole buffer. |

### Viewer (bottom-right pane)

| Binding | Action |
| :--- | :--- |
| `j` / `k` | Scroll rows |
| `h` / `l` | Scroll columns |

---

## Configuration

`lazysql` stores its data under your OS config directory (`$XDG_CONFIG_HOME` on Linux, typically `~/.config`):

```
~/.config/lazysql/
├── connections.json        # saved connections
└── sessions/
    └── session-<pid>.log   # live session log
```

`connections.json` is plain JSON. You can hand-edit it; one entry per connection name.

---

## Roadmap

- Additional drivers (MySQL, SQLite, MSSQL).
- Query history / re-run.
- Configurable theming.

PRs welcome.

---

## License

See [LICENSE](LICENSE).
