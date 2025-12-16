package collector

import (
	"net"
	"strings"
)

// Collector interface defines methods for collecting connection data
type Collector interface {
	GetConnections() ([]Connection, error)
}

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

func guessNetworkInterface(addr string) string {
	if addr == "127.0.0.1" || addr == "::1" {
		return "lo"
	}

	ip := net.ParseIP(addr)
	if ip == nil {
		return ""
	}

	if ip.IsLoopback() {
		return "lo"
	}

	// default interface name varies by OS but we return a generic value
	// actual interface detection would require routing table analysis
	return ""
}

func simplifyIPv6(addr string) string {
	parts := strings.Split(addr, ":")
	for i, part := range parts {
		// parse as hex then format back to remove leading zeros
		var val int64
		for _, c := range part {
			val = val*16 + int64(hexCharToInt(c))
		}
		parts[i] = formatHex(val)
	}
	return strings.Join(parts, ":")
}

func hexCharToInt(c rune) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c - 'a' + 10)
	case c >= 'A' && c <= 'F':
		return int(c - 'A' + 10)
	default:
		return 0
	}
}

func formatHex(val int64) string {
	if val == 0 {
		return "0"
	}
	const hexDigits = "0123456789abcdef"
	var result []byte
	for val > 0 {
		result = append([]byte{hexDigits[val%16]}, result...)
		val /= 16
	}
	return string(result)
}
