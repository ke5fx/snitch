# snitch

a friendlier `ss` / `netstat` for humans. inspect network connections with a clean tui or styled tables.

## install

```bash
go install github.com/karol-broda/snitch@latest
```

## quick start

```bash
snitch              # launch interactive tui
snitch -l           # tui showing only listening sockets
snitch ls           # print styled table and exit
snitch ls -l        # listening sockets only
snitch ls -t -e     # tcp established connections
snitch ls -p        # plain output (parsable)
```

## commands

### `snitch` / `snitch top`

interactive tui with live-updating connection list.

```bash
snitch                  # all connections
snitch -l               # listening only
snitch -t               # tcp only
snitch -e               # established only
snitch -i 2s            # 2 second refresh interval
```

**keybindings:**

```
j/k, ↑/↓      navigate
g/G           top/bottom
t/u           toggle tcp/udp
l/e/o         toggle listen/established/other
s/S           cycle sort / reverse
/             search
enter         connection details
?             help
q             quit
```

### `snitch ls`

one-shot table output. uses a pager automatically if output exceeds terminal height.

```bash
snitch ls               # styled table (default)
snitch ls -l            # listening only
snitch ls -t -l         # tcp listeners
snitch ls -e            # established only
snitch ls -p            # plain/parsable output
snitch ls -o json       # json output
snitch ls -o csv        # csv output
snitch ls -n            # numeric (no dns resolution)
snitch ls --no-headers  # omit headers
```

### `snitch json`

json output for scripting.

```bash
snitch json
snitch json -l
```

### `snitch watch`

stream json frames at an interval.

```bash
snitch watch -i 1s | jq '.count'
snitch watch -l -i 500ms
```

## filters

shortcut flags work on all commands:

```
-t, --tcp           tcp only
-u, --udp           udp only
-l, --listen        listening sockets
-e, --established   established connections
-4, --ipv4          ipv4 only
-6, --ipv6          ipv6 only
-n, --numeric       no dns resolution
```

for more specific filtering, use `key=value` syntax with `ls`:

```bash
snitch ls proto=tcp state=listen
snitch ls pid=1234
snitch ls proc=nginx
snitch ls lport=443
snitch ls contains=google
```

## output

styled table (default):

```
  ╭─────────────────┬───────┬───────┬─────────────┬─────────────────┬────────╮
  │ PROCESS         │ PID   │ PROTO │ STATE       │ LADDR           │ LPORT  │
  ├─────────────────┼───────┼───────┼─────────────┼─────────────────┼────────┤
  │ nginx           │ 1234  │ tcp   │ LISTEN      │ *               │ 80     │
  │ postgres        │ 5678  │ tcp   │ LISTEN      │ 127.0.0.1       │ 5432   │
  ╰─────────────────┴───────┴───────┴─────────────┴─────────────────┴────────╯
  2 connections
```

plain output (`-p`):

```
PROCESS    PID    PROTO   STATE    LADDR       LPORT
nginx      1234   tcp     LISTEN   *           80
postgres   5678   tcp     LISTEN   127.0.0.1   5432
```

## configuration

optional config file at `~/.config/snitch/snitch.toml`:

```toml
[defaults]
numeric = false
theme = "auto"
```

## requirements

- linux (reads from `/proc/net/*`)
- root or `CAP_NET_ADMIN` for full process info
