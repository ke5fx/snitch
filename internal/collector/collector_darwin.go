//go:build darwin

package collector

/*
#include <libproc.h>
#include <sys/proc_info.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <pwd.h>
#include <stdlib.h>
#include <string.h>

// get process name by pid
static int get_proc_name(int pid, char *name, int namelen) {
    return proc_name(pid, name, namelen);
}

// get process path by pid
static int get_proc_path(int pid, char *path, int pathlen) {
    return proc_pidpath(pid, path, pathlen);
}

// get uid for a process
static int get_proc_uid(int pid) {
    struct proc_bsdinfo info;
    int ret = proc_pidinfo(pid, PROC_PIDTBSDINFO, 0, &info, sizeof(info));
    if (ret <= 0) {
        return -1;
    }
    return info.pbi_uid;
}

// get username from uid
static char* get_username(int uid) {
    struct passwd *pw = getpwuid(uid);
    if (pw == NULL) {
        return NULL;
    }
    return pw->pw_name;
}
*/
import "C"

import (
	"fmt"
	"net"
	"strconv"
	"time"
	"unsafe"
)

// DefaultCollector implements the Collector interface using libproc on macOS
type DefaultCollector struct{}

// GetConnections fetches all network connections using libproc
func (dc *DefaultCollector) GetConnections() ([]Connection, error) {
	pids, err := listAllPids()
	if err != nil {
		return nil, fmt.Errorf("failed to list pids: %w", err)
	}

	var connections []Connection

	for _, pid := range pids {
		procConns, err := getConnectionsForPid(pid)
		if err != nil {
			continue
		}
		connections = append(connections, procConns...)
	}

	return connections, nil
}

// GetAllConnections returns network connections (Unix sockets not easily available via libproc)
func GetAllConnections() ([]Connection, error) {
	return GetConnections()
}

func listAllPids() ([]int, error) {
	// first call to get buffer size needed
	numPids := C.proc_listpids(C.PROC_ALL_PIDS, 0, nil, 0)
	if numPids <= 0 {
		return nil, fmt.Errorf("proc_listpids failed")
	}

	// allocate buffer
	bufSize := C.int(numPids) * C.int(unsafe.Sizeof(C.int(0)))
	buf := make([]C.int, numPids)

	// get actual pids
	numPids = C.proc_listpids(C.PROC_ALL_PIDS, 0, unsafe.Pointer(&buf[0]), bufSize)
	if numPids <= 0 {
		return nil, fmt.Errorf("proc_listpids failed")
	}

	count := int(numPids) / int(unsafe.Sizeof(C.int(0)))
	pids := make([]int, 0, count)
	for i := 0; i < count; i++ {
		if buf[i] > 0 {
			pids = append(pids, int(buf[i]))
		}
	}

	return pids, nil
}

func getConnectionsForPid(pid int) ([]Connection, error) {
	// get process info first
	procName := getProcessName(pid)
	uid := int(C.get_proc_uid(C.int(pid)))
	user := ""
	if uid >= 0 {
		cUser := C.get_username(C.int(uid))
		if cUser != nil {
			user = C.GoString(cUser)
		} else {
			user = strconv.Itoa(uid)
		}
	}

	// get file descriptors for this process
	bufSize := C.proc_pidinfo(C.int(pid), C.PROC_PIDLISTFDS, 0, nil, 0)
	if bufSize <= 0 {
		return nil, fmt.Errorf("failed to get fd list size")
	}

	buf := make([]byte, bufSize)
	ret := C.proc_pidinfo(C.int(pid), C.PROC_PIDLISTFDS, 0, unsafe.Pointer(&buf[0]), bufSize)
	if ret <= 0 {
		return nil, fmt.Errorf("failed to get fd list")
	}

	fdInfoSize := int(unsafe.Sizeof(C.struct_proc_fdinfo{}))
	numFds := int(ret) / fdInfoSize

	var connections []Connection

	for i := 0; i < numFds; i++ {
		fdInfo := (*C.struct_proc_fdinfo)(unsafe.Pointer(&buf[i*fdInfoSize]))

		// only interested in sockets
		if fdInfo.proc_fdtype != C.PROX_FDTYPE_SOCKET {
			continue
		}

		conn, ok := getSocketInfo(pid, int(fdInfo.proc_fd), procName, uid, user)
		if ok {
			connections = append(connections, conn)
		}
	}

	return connections, nil
}

func getSocketInfo(pid, fd int, procName string, uid int, user string) (Connection, bool) {
	var socketInfo C.struct_socket_fdinfo

	ret := C.proc_pidfdinfo(
		C.int(pid),
		C.int(fd),
		C.PROC_PIDFDSOCKETINFO,
		unsafe.Pointer(&socketInfo),
		C.int(unsafe.Sizeof(socketInfo)),
	)

	if ret <= 0 {
		return Connection{}, false
	}

	// check socket family - only interested in IPv4 and IPv6
	family := socketInfo.psi.soi_family
	if family != C.AF_INET && family != C.AF_INET6 {
		return Connection{}, false
	}

	// check socket type - only TCP and UDP
	sockType := socketInfo.psi.soi_type
	if sockType != C.SOCK_STREAM && sockType != C.SOCK_DGRAM {
		return Connection{}, false
	}

	proto := "tcp"
	if sockType == C.SOCK_DGRAM {
		proto = "udp"
	}

	ipVersion := "IPv4"
	if family == C.AF_INET6 {
		ipVersion = "IPv6"
		proto = proto + "6"
	}

	var laddr, raddr string
	var lport, rport int
	var state string

	if family == C.AF_INET {
		// IPv4
		insi := socketInfo.psi.soi_proto.pri_tcp.tcpsi_ini
		laddr = ipv4ToString(insi.insi_laddr.ina_46.i46a_addr4.s_addr)
		raddr = ipv4ToString(insi.insi_faddr.ina_46.i46a_addr4.s_addr)
		lport = int(ntohs(insi.insi_lport))
		rport = int(ntohs(insi.insi_fport))

		if sockType == C.SOCK_STREAM {
			state = tcpStateToString(int(socketInfo.psi.soi_proto.pri_tcp.tcpsi_state))
		}
	} else {
		// IPv6
		insi := socketInfo.psi.soi_proto.pri_tcp.tcpsi_ini
		laddr = ipv6ToString(insi.insi_laddr.ina_6)
		raddr = ipv6ToString(insi.insi_faddr.ina_6)
		lport = int(ntohs(insi.insi_lport))
		rport = int(ntohs(insi.insi_fport))

		if sockType == C.SOCK_STREAM {
			state = tcpStateToString(int(socketInfo.psi.soi_proto.pri_tcp.tcpsi_state))
		}
	}

	// normalize wildcard addresses
	if laddr == "0.0.0.0" || laddr == "::" {
		laddr = "*"
	}
	if raddr == "0.0.0.0" || raddr == "::" {
		raddr = "*"
	}

	conn := Connection{
		TS:        time.Now(),
		Proto:     proto,
		IPVersion: ipVersion,
		State:     state,
		Laddr:     laddr,
		Lport:     lport,
		Raddr:     raddr,
		Rport:     rport,
		PID:       pid,
		Process:   procName,
		UID:       uid,
		User:      user,
		Interface: guessNetworkInterface(laddr),
	}

	return conn, true
}

func getProcessName(pid int) string {
	var name [256]C.char
	ret := C.get_proc_name(C.int(pid), &name[0], 256)
	if ret <= 0 {
		return ""
	}
	return C.GoString(&name[0])
}

func ipv4ToString(addr C.in_addr_t) string {
	ip := make(net.IP, 4)
	ip[0] = byte(addr)
	ip[1] = byte(addr >> 8)
	ip[2] = byte(addr >> 16)
	ip[3] = byte(addr >> 24)
	return ip.String()
}

func ipv6ToString(addr C.struct_in6_addr) string {
	ip := make(net.IP, 16)
	for i := 0; i < 16; i++ {
		ip[i] = byte(addr.__u6_addr.__u6_addr8[i])
	}

	// check for IPv4-mapped IPv6 addresses
	if ip.To4() != nil {
		return ip.To4().String()
	}

	return ip.String()
}

func ntohs(port C.int) uint16 {
	return uint16((port&0xff)<<8 | (port>>8)&0xff)
}

func tcpStateToString(state int) string {
	states := map[int]string{
		0:  "CLOSED",
		1:  "LISTEN",
		2:  "SYN_SENT",
		3:  "SYN_RECV",
		4:  "ESTABLISHED",
		5:  "CLOSE_WAIT",
		6:  "FIN_WAIT1",
		7:  "CLOSING",
		8:  "LAST_ACK",
		9:  "FIN_WAIT2",
		10: "TIME_WAIT",
	}

	if s, exists := states[state]; exists {
		return s
	}
	return ""
}
