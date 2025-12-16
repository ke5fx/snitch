package collector

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Collector interface defines methods for collecting connection data
type Collector interface {
	GetConnections() ([]Connection, error)
}

// DefaultCollector implements the Collector interface using /proc
type DefaultCollector struct{}

// Global collector instance (can be overridden for testing)
var globalCollector Collector = &DefaultCollector{}

// SetCollector sets the global collector instance
func SetCollector(collector Collector) {
	globalCollector = collector
}

// GetCollector returns the current global collector instance
func GetCollector() Collector {
	return globalCollector
}

// GetConnections fetches all network connections using the global collector
func GetConnections() ([]Connection, error) {
	return globalCollector.GetConnections()
}

// GetConnections fetches all network connections by parsing /proc files.
func (dc *DefaultCollector) GetConnections() ([]Connection, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("proc-based collector only supports Linux")
	}

	// Build map of inode -> process info by scanning /proc
	inodeMap, err := buildInodeToProcessMap()
	if err != nil {
		return nil, fmt.Errorf("failed to build inode map: %w", err)
	}

	var connections []Connection

	// Parse TCP connections
	tcpConns, err := parseProcNet("/proc/net/tcp", "tcp", 4, inodeMap)
	if err == nil {
		connections = append(connections, tcpConns...)
	}

	tcpConns6, err := parseProcNet("/proc/net/tcp6", "tcp6", 6, inodeMap)
	if err == nil {
		connections = append(connections, tcpConns6...)
	}

	// Parse UDP connections
	udpConns, err := parseProcNet("/proc/net/udp", "udp", 4, inodeMap)
	if err == nil {
		connections = append(connections, udpConns...)
	}

	udpConns6, err := parseProcNet("/proc/net/udp6", "udp6", 6, inodeMap)
	if err == nil {
		connections = append(connections, udpConns6...)
	}

	return connections, nil
}

// GetAllConnections returns both network and Unix domain socket connections
func GetAllConnections() ([]Connection, error) {
	// Get network connections
	networkConns, err := GetConnections()
	if err != nil {
		return nil, err
	}

	// Get Unix sockets (only on Linux)
	if runtime.GOOS == "linux" {
		unixConns, err := GetUnixSockets()
		if err == nil {
			networkConns = append(networkConns, unixConns...)
		}
	}

	return networkConns, nil
}

func FilterConnections(conns []Connection, filters FilterOptions) []Connection {
	if filters.IsEmpty() {
		return conns
	}

	filtered := make([]Connection, 0)
	for _, conn := range conns {
		if filters.Matches(conn) {
			filtered = append(filtered, conn)
		}
	}
	return filtered
}

// processInfo holds information about a process
type processInfo struct {
	pid     int
	command string
	uid     int
	user    string
}

// buildInodeToProcessMap scans /proc to map socket inodes to processes
func buildInodeToProcessMap() (map[int64]*processInfo, error) {
	inodeMap := make(map[int64]*processInfo)

	procDir, err := os.Open("/proc")
	if err != nil {
		return nil, err
	}
	defer procDir.Close()

	entries, err := procDir.Readdir(-1)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// check if directory name is a number (pid)
		pidStr := entry.Name()
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		// get process info
		procInfo, err := getProcessInfo(pid)
		if err != nil {
			continue
		}

		// scan /proc/[pid]/fd/ for socket file descriptors
		fdDir := filepath.Join("/proc", pidStr, "fd")
		fdEntries, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}

		for _, fdEntry := range fdEntries {
			fdPath := filepath.Join(fdDir, fdEntry.Name())
			link, err := os.Readlink(fdPath)
			if err != nil {
				continue
			}

			// socket inodes look like: socket:[12345]
			if strings.HasPrefix(link, "socket:[") && strings.HasSuffix(link, "]") {
				inodeStr := link[8 : len(link)-1]
				inode, err := strconv.ParseInt(inodeStr, 10, 64)
				if err != nil {
					continue
				}
				inodeMap[inode] = procInfo
			}
		}
	}

	return inodeMap, nil
}

// getProcessInfo reads process information from /proc/[pid]/
func getProcessInfo(pid int) (*processInfo, error) {
	info := &processInfo{pid: pid}

	// prefer /proc/[pid]/comm as it's always just the command name
	commPath := filepath.Join("/proc", strconv.Itoa(pid), "comm")
	commData, err := os.ReadFile(commPath)
	if err == nil && len(commData) > 0 {
		info.command = strings.TrimSpace(string(commData))
	}

	// if comm is not available, try cmdline
	if info.command == "" {
		cmdlinePath := filepath.Join("/proc", strconv.Itoa(pid), "cmdline")
		cmdlineData, err := os.ReadFile(cmdlinePath)
		if err != nil {
			return nil, err
		}

		// cmdline is null-separated, take first part
		if len(cmdlineData) > 0 {
			parts := bytes.Split(cmdlineData, []byte{0})
			if len(parts) > 0 && len(parts[0]) > 0 {
				fullPath := string(parts[0])
				// extract basename from full path
				baseName := filepath.Base(fullPath)
				// if basename contains spaces (single-string cmdline), take first word
				if strings.Contains(baseName, " ") {
					baseName = strings.Fields(baseName)[0]
				}
				info.command = baseName
			}
		}
	}

	// read UID from /proc/[pid]/status
	statusPath := filepath.Join("/proc", strconv.Itoa(pid), "status")
	statusFile, err := os.Open(statusPath)
	if err != nil {
		return info, nil
	}
	defer statusFile.Close()

	scanner := bufio.NewScanner(statusFile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				uid, err := strconv.Atoi(fields[1])
				if err == nil {
					info.uid = uid
					// get username from uid
					u, err := user.LookupId(strconv.Itoa(uid))
					if err == nil {
						info.user = u.Username
					} else {
						info.user = strconv.Itoa(uid)
					}
				}
			}
			break
		}
	}

	return info, nil
}

// parseProcNet parses a /proc/net/tcp or /proc/net/udp file
func parseProcNet(path, proto string, ipVersion int, inodeMap map[int64]*processInfo) ([]Connection, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var connections []Connection
	scanner := bufio.NewScanner(file)

	// skip header
	scanner.Scan()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		// parse local address and port
		localAddr, localPort, err := parseHexAddr(fields[1])
		if err != nil {
			continue
		}

		// parse remote address and port
		remoteAddr, remotePort, err := parseHexAddr(fields[2])
		if err != nil {
			continue
		}

		// parse state (field 3)
		stateHex := fields[3]
		state := parseState(stateHex, proto)

		// parse inode (field 9)
		inode, _ := strconv.ParseInt(fields[9], 10, 64)

		conn := Connection{
			TS:        time.Now(),
			Proto:     proto,
			IPVersion: fmt.Sprintf("IPv%d", ipVersion),
			State:     state,
			Laddr:     localAddr,
			Lport:     localPort,
			Raddr:     remoteAddr,
			Rport:     remotePort,
			Inode:     inode,
		}

		// add process info if available
		if procInfo, exists := inodeMap[inode]; exists {
			conn.PID = procInfo.pid
			conn.Process = procInfo.command
			conn.UID = procInfo.uid
			conn.User = procInfo.user
		}

		// determine interface
		conn.Interface = guessNetworkInterface(localAddr, nil)

		connections = append(connections, conn)
	}

	return connections, scanner.Err()
}

// parseState converts hex state value to string
func parseState(hexState, proto string) string {
	state, err := strconv.ParseInt(hexState, 16, 32)
	if err != nil {
		return ""
	}

	// TCP states
	tcpStates := map[int64]string{
		0x01: "ESTABLISHED",
		0x02: "SYN_SENT",
		0x03: "SYN_RECV",
		0x04: "FIN_WAIT1",
		0x05: "FIN_WAIT2",
		0x06: "TIME_WAIT",
		0x07: "CLOSE",
		0x08: "CLOSE_WAIT",
		0x09: "LAST_ACK",
		0x0A: "LISTEN",
		0x0B: "CLOSING",
	}

	if strings.HasPrefix(proto, "tcp") {
		if s, exists := tcpStates[state]; exists {
			return s
		}
	} else {
		// UDP doesn't have states in the same way
		if state == 0x07 {
			return "CLOSE"
		}
		return ""
	}

	return ""
}

// parseHexAddr parses hex-encoded address:port from /proc/net files
func parseHexAddr(hexAddr string) (string, int, error) {
	parts := strings.Split(hexAddr, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid address format")
	}

	hexIP := parts[0]

	// parse hex port
	port, err := strconv.ParseInt(parts[1], 16, 32)
	if err != nil {
		return "", 0, err
	}

	if len(hexIP) == 8 {
		// IPv4 (stored in little-endian)
		ip1, _ := strconv.ParseInt(hexIP[6:8], 16, 32)
		ip2, _ := strconv.ParseInt(hexIP[4:6], 16, 32)
		ip3, _ := strconv.ParseInt(hexIP[2:4], 16, 32)
		ip4, _ := strconv.ParseInt(hexIP[0:2], 16, 32)
		addr := fmt.Sprintf("%d.%d.%d.%d", ip1, ip2, ip3, ip4)

		// handle wildcard address
		if addr == "0.0.0.0" {
			addr = "*"
		}

		return addr, int(port), nil
	} else if len(hexIP) == 32 {
		// IPv6 (stored in little-endian per 32-bit word)
		var ipv6Parts []string
		for i := 0; i < 32; i += 8 {
			word := hexIP[i : i+8]
			// reverse byte order within each 32-bit word
			p1 := word[6:8] + word[4:6] + word[2:4] + word[0:2]
			ipv6Parts = append(ipv6Parts, p1)
		}

		// convert to standard IPv6 notation
		fullAddr := strings.Join(ipv6Parts, "")
		var formatted []string
		for i := 0; i < len(fullAddr); i += 4 {
			formatted = append(formatted, fullAddr[i:i+4])
		}
		addr := strings.Join(formatted, ":")

		// simplify IPv6 address
		addr = simplifyIPv6(addr)

		// handle wildcard address
		if addr == "::" || addr == "0:0:0:0:0:0:0:0" {
			addr = "*"
		}

		return addr, int(port), nil
	}

	return "", 0, fmt.Errorf("unsupported address format")
}

// simplifyIPv6 simplifies IPv6 address notation
func simplifyIPv6(addr string) string {
	// remove leading zeros from each group
	parts := strings.Split(addr, ":")
	for i, part := range parts {
		// convert to int and back to remove leading zeros
		val, err := strconv.ParseInt(part, 16, 64)
		if err == nil {
			parts[i] = strconv.FormatInt(val, 16)
		}
	}
	return strings.Join(parts, ":")
}

func guessNetworkInterface(addr string, interfaces map[string]string) string {
	// Simple heuristic - try to match common interface patterns
	if addr == "127.0.0.1" || addr == "::1" {
		return "lo"
	}

	// Check if it's a private network address
	ip := net.ParseIP(addr)
	if ip != nil {
		if ip.IsLoopback() {
			return "lo"
		}
		// More sophisticated interface detection would require routing table analysis
		// For now, return a placeholder
		if ip.To4() != nil {
			return "eth0" // Common default for IPv4
		} else {
			return "eth0" // Common default for IPv6
		}
	}

	return ""
}

// Add Unix socket support
func GetUnixSockets() ([]Connection, error) {
	connections := []Connection{}

	// Parse /proc/net/unix for Unix domain sockets
	file, err := os.Open("/proc/net/unix")
	if err != nil {
		return connections, nil // silently fail on non-Linux systems
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Skip header
	scanner.Scan()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}

		// Parse Unix socket information
		inode, _ := strconv.ParseInt(fields[6], 10, 64)
		path := ""
		if len(fields) > 7 {
			path = fields[7]
		}

		conn := Connection{
			TS:        time.Now(),
			Proto:     "unix",
			Laddr:     path,
			Raddr:     "",
			State:     "CONNECTED", // Simplified
			Inode:     inode,
			Interface: "unix",
		}

		connections = append(connections, conn)
	}

	return connections, nil
}

