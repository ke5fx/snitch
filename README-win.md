# snitch-win

A friendlier `netstat` for humans, ported to Windows in Visual C++.

## Features
- **Interactive TUI:** Live-updating connection list with flicker-free rendering.
- **Color-Coded Output:** 
  - **Bright White**: ESTABLISHED TCP connections.
  - **Cyan**: UDP processes.
  - **Dark Gray**: Other TCP states (LISTEN, CLOSE_WAIT, etc.).
- **Deduplication:** Unique process mode to show only one entry per executable.
- **Full Path Mapping:** Option to display the complete executable path.
- **No External Dependencies:** Uses standard Win32 APIs (`iphlpapi`, `psapi`).
- **Modern UI:** UTF-8 box-drawing characters and ANSI color support.

## Building
Run the provided `build.bat` script from a Visual Studio Developer Command Prompt:
```batch
build.bat
```

## Usage
### Interactive Mode (Default)
```batch
snitch.exe
```
**Keybindings in TUI:**
- `q`: Quit
- `t`: Toggle TCP filter
- `u`: Toggle UDP filter
- `l`: Toggle Listen filter
- `e`: Toggle Established filter
- `U`: Toggle Unique process mode (one line per .exe)
- `f`: Toggle Full executable paths

### One-shot Mode
```batch
snitch.exe ls [options]
```

### Options
- `-t`, `--tcp`: TCP only.
- `-u`, `--udp`: UDP only.
- `-l`, `--listen`: Listening sockets only.
- `-e`, `--established`: Established connections only.
- `-4`, `--ipv4`: IPv4 only.
- `-6`, `--ipv6`: IPv6 only.
- `-n`, `--numeric`: No DNS resolution (default).
- `-U`, `--unique`: List each process executable only once.
- `-f`, `--full-path`: Show full process executable path.
- `-h`, `--help`: Show help.