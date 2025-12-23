#define WIN32_LEAN_AND_MEAN
#define NOMINMAX
#include <windows.h>
#include <winsock2.h>
#include <ws2tcpip.h>
#include <iphlpapi.h>
#include <psapi.h>
#include <conio.h>
#include <iostream>
#include <vector>
#include <string>
#include <iomanip>
#include <algorithm>
#include <chrono>
#include <thread>
#include <set>

#pragma comment(lib, "iphlpapi.lib")
#pragma comment(lib, "ws2_32.lib")

enum class Proto { TCP, UDP };
enum class IpVer { IPv4, IPv6 };

struct Connection {
    std::string processName;
    DWORD pid;
    Proto proto;
    IpVer ipVer;
    std::string state;
    std::string localAddr;
    int localPort;
    std::string remoteAddr;
    int remotePort;
};

struct Filters {
    bool tcp = true;
    bool udp = true;
    bool listen = false;
    bool established = false;
    bool ipv4 = true;
    bool ipv6 = true;
    bool numeric = true;
    bool unique = false;
    bool fullPath = false;
};

std::string WStringToString(const std::wstring& wstr) {
    if (wstr.empty()) return std::string();
    int size_needed = WideCharToMultiByte(CP_UTF8, 0, &wstr[0], (int)wstr.size(), NULL, 0, NULL, NULL);
    std::string strTo(size_needed, 0);
    WideCharToMultiByte(CP_UTF8, 0, &wstr[0], (int)wstr.size(), &strTo[0], size_needed, NULL, NULL);
    return strTo;
}

std::string GetProcessName(DWORD pid, bool fullPath = false) {
    if (pid == 0) return "Idle";
    if (pid == 4) return "System";

    HANDLE hProcess = OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, FALSE, pid);
    if (hProcess) {
        wchar_t szPath[MAX_PATH];
        DWORD dwSize = MAX_PATH;
        if (QueryFullProcessImageNameW(hProcess, 0, szPath, &dwSize)) {
            std::wstring path(szPath);
            if (!fullPath) {
                size_t pos = path.find_last_of(L"\\");
                if (pos == std::wstring::npos) pos = path.find_last_of(L"/");
                path = (pos == std::wstring::npos) ? path : path.substr(pos + 1);
            }
            CloseHandle(hProcess);
            return WStringToString(path);
        }
        CloseHandle(hProcess);
    }
    return "Unknown";
}

std::string IpToString(DWORD addr) {
    IN_ADDR inAddr;
    inAddr.s_addr = addr;
    char buf[INET_ADDRSTRLEN];
    inet_ntop(AF_INET, &inAddr, buf, INET_ADDRSTRLEN);
    return std::string(buf);
}

std::string Ip6ToString(UCHAR addr[16]) {
    IN6_ADDR inAddr;
    memcpy(&inAddr, addr, 16);
    char buf[INET6_ADDRSTRLEN];
    inet_ntop(AF_INET6, &inAddr, buf, INET6_ADDRSTRLEN);
    return std::string(buf);
}

std::string TcpStateToString(DWORD state) {
    switch (state) {
        case MIB_TCP_STATE_CLOSED: return "CLOSED";
        case MIB_TCP_STATE_LISTEN: return "LISTEN";
        case MIB_TCP_STATE_SYN_SENT: return "SYN_SENT";
        case MIB_TCP_STATE_SYN_RCVD: return "SYN_RCVD";
        case MIB_TCP_STATE_ESTAB: return "ESTABLISHED";
        case MIB_TCP_STATE_FIN_WAIT1: return "FIN_WAIT1";
        case MIB_TCP_STATE_FIN_WAIT2: return "FIN_WAIT2";
        case MIB_TCP_STATE_CLOSE_WAIT: return "CLOSE_WAIT";
        case MIB_TCP_STATE_CLOSING: return "CLOSING";
        case MIB_TCP_STATE_LAST_ACK: return "LAST_ACK";
        case MIB_TCP_STATE_TIME_WAIT: return "TIME_WAIT";
        case MIB_TCP_STATE_DELETE_TCB: return "DELETE_TCB";
        default: return "UNKNOWN";
    }
}

void GetTcpConnections(std::vector<Connection>& connections, const Filters& f) {
    if (!f.tcp) return;
    if (f.ipv4) {
        ULONG size = 0;
        GetExtendedTcpTable(NULL, &size, FALSE, AF_INET, TCP_TABLE_OWNER_PID_ALL, 0);
        std::vector<BYTE> buffer(size);
        if (GetExtendedTcpTable(buffer.data(), &size, FALSE, AF_INET, TCP_TABLE_OWNER_PID_ALL, 0) == NO_ERROR) {
            PMIB_TCPTABLE_OWNER_PID table = (PMIB_TCPTABLE_OWNER_PID)buffer.data();
            for (DWORD i = 0; i < table->dwNumEntries; i++) {
                if (f.listen && table->table[i].dwState != MIB_TCP_STATE_LISTEN) continue;
                if (f.established && table->table[i].dwState != MIB_TCP_STATE_ESTAB) continue;
                Connection conn;
                conn.pid = table->table[i].dwOwningPid;
                conn.processName = GetProcessName(conn.pid, f.fullPath);
                conn.proto = Proto::TCP;
                conn.ipVer = IpVer::IPv4;
                conn.state = TcpStateToString(table->table[i].dwState);
                conn.localAddr = IpToString(table->table[i].dwLocalAddr);
                conn.localPort = ntohs((u_short)table->table[i].dwLocalPort);
                conn.remoteAddr = IpToString(table->table[i].dwRemoteAddr);
                conn.remotePort = ntohs((u_short)table->table[i].dwRemotePort);
                connections.push_back(conn);
            }
        }
    }
    if (f.ipv6) {
        ULONG size = 0;
        GetExtendedTcpTable(NULL, &size, FALSE, AF_INET6, TCP_TABLE_OWNER_PID_ALL, 0);
        std::vector<BYTE> buffer(size);
        if (GetExtendedTcpTable(buffer.data(), &size, FALSE, AF_INET6, TCP_TABLE_OWNER_PID_ALL, 0) == NO_ERROR) {
            PMIB_TCP6TABLE_OWNER_PID table = (PMIB_TCP6TABLE_OWNER_PID)buffer.data();
            for (DWORD i = 0; i < table->dwNumEntries; i++) {
                if (f.listen && table->table[i].dwState != MIB_TCP_STATE_LISTEN) continue;
                if (f.established && table->table[i].dwState != MIB_TCP_STATE_ESTAB) continue;
                Connection conn;
                conn.pid = table->table[i].dwOwningPid;
                conn.processName = GetProcessName(conn.pid, f.fullPath);
                conn.proto = Proto::TCP;
                conn.ipVer = IpVer::IPv6;
                conn.state = TcpStateToString(table->table[i].dwState);
                conn.localAddr = Ip6ToString(table->table[i].ucLocalAddr);
                conn.localPort = ntohs((u_short)table->table[i].dwLocalPort);
                conn.remoteAddr = Ip6ToString(table->table[i].ucRemoteAddr);
                conn.remotePort = ntohs((u_short)table->table[i].dwRemotePort);
                connections.push_back(conn);
            }
        }
    }
}

void GetUdpConnections(std::vector<Connection>& connections, const Filters& f) {
    if (!f.udp || f.established) return;
    if (f.ipv4) {
        ULONG size = 0;
        GetExtendedUdpTable(NULL, &size, FALSE, AF_INET, UDP_TABLE_OWNER_PID, 0);
        std::vector<BYTE> buffer(size);
        if (GetExtendedUdpTable(buffer.data(), &size, FALSE, AF_INET, UDP_TABLE_OWNER_PID, 0) == NO_ERROR) {
            PMIB_UDPTABLE_OWNER_PID table = (PMIB_UDPTABLE_OWNER_PID)buffer.data();
            for (DWORD i = 0; i < table->dwNumEntries; i++) {
                Connection conn;
                conn.pid = table->table[i].dwOwningPid;
                conn.processName = GetProcessName(conn.pid, f.fullPath);
                conn.proto = Proto::UDP;
                conn.ipVer = IpVer::IPv4;
                conn.state = "";
                conn.localAddr = IpToString(table->table[i].dwLocalAddr);
                conn.localPort = ntohs((u_short)table->table[i].dwLocalPort);
                conn.remoteAddr = "*";
                conn.remotePort = 0;
                connections.push_back(conn);
            }
        }
    }
    if (f.ipv6) {
        ULONG size = 0;
        GetExtendedUdpTable(NULL, &size, FALSE, AF_INET6, UDP_TABLE_OWNER_PID, 0);
        std::vector<BYTE> buffer(size);
        if (GetExtendedUdpTable(buffer.data(), &size, FALSE, AF_INET6, UDP_TABLE_OWNER_PID, 0) == NO_ERROR) {
            PMIB_UDP6TABLE_OWNER_PID table = (PMIB_UDP6TABLE_OWNER_PID)buffer.data();
            for (DWORD i = 0; i < table->dwNumEntries; i++) {
                Connection conn;
                conn.pid = table->table[i].dwOwningPid;
                conn.processName = GetProcessName(conn.pid, f.fullPath);
                conn.proto = Proto::UDP;
                conn.ipVer = IpVer::IPv6;
                conn.state = "";
                conn.localAddr = Ip6ToString(table->table[i].ucLocalAddr);
                conn.localPort = ntohs((u_short)table->table[i].dwLocalPort);
                conn.remoteAddr = "*";
                conn.remotePort = 0;
                connections.push_back(conn);
            }
        }
    }
}

std::string Repeat(std::string s, int n) {
    std::string res;
    for (int i = 0; i < n; i++) res += s;
    return res;
}

void PrintTable(const std::vector<Connection>& connections, int maxHeight = -1, bool uniqueMode = false) {
    if (connections.empty()) {
        std::cout << "0 processes/connections" << std::endl;
        return;
    }

    size_t wProc = 7, wPid = 3, wProto = 5, wState = 5, wLAddr = 5, wLPort = 5;
    for (const auto& c : connections) {
        wProc = std::max(wProc, c.processName.length());
        wPid = std::max(wPid, std::to_string(c.pid).length());
        wState = std::max(wState, c.state.length());
        wLAddr = std::max(wLAddr, c.localAddr.length());
        wLPort = std::max(wLPort, std::to_string(c.localPort).length());
    }

    // Top border
    std::cout << "╭─" << Repeat("─", (int)wProc) << "─┬─" << Repeat("─", (int)wPid) << "─┬─" << Repeat("─", (int)wProto) << "─┬─" << Repeat("─", (int)wState) << "─┬─" << Repeat("─", (int)wLAddr) << "─┬─" << Repeat("─", (int)wLPort) << "─╮\033[K" << std::endl;
    
    // Header
    std::cout << "│ " << std::left << std::setw((int)wProc) << "PROCESS"
            << " │ " << std::setw((int)wPid) << "PID"
            << " │ " << std::setw((int)wProto) << "PROTO"
            << " │ " << std::setw((int)wState) << "STATE"
            << " │ " << std::setw((int)wLAddr) << "LADDR"
            << " │ " << std::setw((int)wLPort) << "LPORT" << " │\033[K" << std::endl;

    // Separator
    std::cout << "├─" << Repeat("─", (int)wProc) << "─┼─" << Repeat("─", (int)wPid) << "─┼─" << Repeat("─", (int)wProto) << "─┼─" << Repeat("─", (int)wState) << "─┼─" << Repeat("─", (int)wLAddr) << "─┼─" << Repeat("─", (int)wLPort) << "─┤\033[K" << std::endl;

    int count = 0;
    for (const auto& c : connections) {
        if (maxHeight > 0 && count >= maxHeight - 5) break;
        std::string displayAddr = c.localAddr;
        if (displayAddr == "0.0.0.0" || displayAddr == "::") displayAddr = "*";

        std::string colorPrefix = "";
        if (c.proto == Proto::UDP) {
            colorPrefix = "\033[36m"; // Cyan
        } else {
            if (c.state == "ESTABLISHED") {
                colorPrefix = "\033[97m"; // Bright White
            } else {
                colorPrefix = "\033[90m"; // Dark Gray
            }
        }

        std::cout << "│ " << colorPrefix << std::left << std::setw((int)wProc) << (c.processName.empty() ? "?" : c.processName) << "\033[0m"
                << " │ " << colorPrefix << std::right << std::setw((int)wPid) << c.pid << "\033[0m"
                << " │ " << colorPrefix << std::left << std::setw((int)wProto) << (c.proto == Proto::TCP ? "tcp" : "udp") << "\033[0m"
                << " │ " << colorPrefix << std::setw((int)wState) << (c.state.empty() ? "-" : c.state) << "\033[0m"
                << " │ " << colorPrefix << std::setw((int)wLAddr) << displayAddr << "\033[0m"
                << " │ " << colorPrefix << std::right << std::setw((int)wLPort) << c.localPort << "\033[0m" << " │\033[K" << std::endl;
        count++;
    }

    // Bottom border
    std::cout << "╰─" << Repeat("─", (int)wProc) << "─┴─" << Repeat("─", (int)wPid) << "─┴─" << Repeat("─", (int)wProto) << "─┴─" << Repeat("─", (int)wState) << "─┴─" << Repeat("─", (int)wLAddr) << "─┴─" << Repeat("─", (int)wLPort) << "─╯\033[K" << std::endl;
    if (uniqueMode)
        std::cout << connections.size() << " unique processes";
    else
        std::cout << connections.size() << " connections";

    if (maxHeight > 0 && (int)connections.size() > count) std::cout << " (truncated)";
    std::cout << "\033[K" << std::endl;
}

void PrintHelp() {
    std::cout << "snitch-win: a friendlier netstat for humans (Windows version)" << std::endl;
    std::cout << "Usage: snitch [ls] [options]" << std::endl;
    std::cout << "Options:" << std::endl;
    std::cout << "  -t, --tcp          TCP only" << std::endl;
    std::cout << "  -u, --udp          UDP only" << std::endl;
    std::cout << "  -l, --listen       Listening sockets only" << std::endl;
    std::cout << "  -e, --established  Established connections only" << std::endl;
    std::cout << "  -4, --ipv4         IPv4 only" << std::endl;
    std::cout << "  -6, --ipv6         IPv6 only" << std::endl;
    std::cout << "  -n, --numeric      No DNS resolution (default)" << std::endl;
    std::cout << "  -U, --unique       List each process only once" << std::endl;
    std::cout << "  -f, --full-path    Show full process executable path" << std::endl;
    std::cout << "  -h, --help         Show this help" << std::endl;
    std::cout << "\nInteractive Mode (TUI) Keybindings:" << std::endl;
    std::cout << "  q                  Quit" << std::endl;
    std::cout << "  t                  Toggle TCP" << std::endl;
    std::cout << "  u                  Toggle UDP" << std::endl;
    std::cout << "  l                  Toggle Listen" << std::endl;
    std::cout << "  e                  Toggle Established" << std::endl;
    std::cout << "  U                  Toggle Unique processes" << std::endl;
    std::cout << "  f                  Toggle Full paths" << std::endl;
}

void RunTui(Filters f) {
    bool quit = false;
    while (!quit) {
        if (_kbhit()) {
            int ch = _getch();
            if (ch == 'q' || ch == 'Q') quit = true;
            else if (ch == 't' || ch == 'T') f.tcp = !f.tcp;
            else if (ch == 'u' || ch == 'U') f.udp = !f.udp;
            else if (ch == 'l' || ch == 'L') { f.listen = !f.listen; if (f.listen) f.established = false; }
            else if (ch == 'e' || ch == 'E') { f.established = !f.established; if (f.established) f.listen = false; }
            else if (ch == 'U') f.unique = !f.unique;
            else if (ch == 'f' || ch == 'F') f.fullPath = !f.fullPath;
        }

        if (quit) break;

        std::vector<Connection> connections;
        GetTcpConnections(connections, f);
        GetUdpConnections(connections, f);
    
        std::sort(connections.begin(), connections.end(), [](const Connection& a, const Connection& b) {
            if (a.processName != b.processName) return a.processName < b.processName;
            return a.pid < b.pid;
        });

        if (f.unique) {
            auto last = std::unique(connections.begin(), connections.end(), [](const Connection& a, const Connection& b) {
                return a.processName == b.processName;
            });
            connections.erase(last, connections.end());
        }
    
        // Get console height
        CONSOLE_SCREEN_BUFFER_INFO csbi;
        GetConsoleScreenBufferInfo(GetStdHandle(STD_OUTPUT_HANDLE), &csbi);
        int height = csbi.srWindow.Bottom - csbi.srWindow.Top + 1;

        std::cout << "\033[H"; // Cursor to Home
        std::cout << "snitch-win [T:" << (f.tcp?"ON":"OFF") << " U:" << (f.udp?"ON":"OFF") << " L:" << (f.listen?"ON":"OFF") << " E:" << (f.established?"ON":"OFF") << " Uniq:" << (f.unique?"ON":"OFF") << " Path:" << (f.fullPath?"ON":"OFF") << "] - 'q' quit\033[K" << std::endl;
        PrintTable(connections, height - 2, f.unique);
        std::cout << "\033[J"; // Clear remaining lines to bottom of screen

        std::this_thread::sleep_for(std::chrono::milliseconds(1000));
    }
}

int main(int argc, char* argv[]) {
    SetConsoleOutputCP(CP_UTF8);

    // Enable Virtual Terminal Processing
    HANDLE hOut = GetStdHandle(STD_OUTPUT_HANDLE);
    DWORD dwMode = 0;
    GetConsoleMode(hOut, &dwMode);
    SetConsoleMode(hOut, dwMode | ENABLE_VIRTUAL_TERMINAL_PROCESSING);

    Filters f;
    bool explicitProto = false;
    bool explicitIp = false;
    bool lsMode = false;

    for (int i = 1; i < argc; i++) {
        std::string arg = argv[i];
        if (arg == "ls") { lsMode = true; continue; }
        if (arg == "-t" || arg == "--tcp") { f.tcp = true; if (!explicitProto) { f.udp = false; explicitProto = true; } }
        else if (arg == "-u" || arg == "--udp") { f.udp = true; if (!explicitProto) { f.tcp = false; explicitProto = true; } }
        else if (arg == "-l" || arg == "--listen") { f.listen = true; }
        else if (arg == "-e" || arg == "--established") { f.established = true; }
        else if (arg == "-4" || arg == "--ipv4") { f.ipv4 = true; if (!explicitIp) { f.ipv6 = false; explicitIp = true; } }
        else if (arg == "-6" || arg == "--ipv6") { f.ipv6 = true; if (!explicitIp) { f.ipv4 = false; explicitIp = true; } }
        else if (arg == "-n" || arg == "--numeric") { f.numeric = true; }
        else if (arg == "-U" || arg == "--unique") { f.unique = true; }
        else if (arg == "-f" || arg == "--full-path") { f.fullPath = true; }
        else if (arg == "-h" || arg == "--help") { PrintHelp(); return 0; }
    }

    WSADATA wsaData;
    WSAStartup(MAKEWORD(2, 2), &wsaData);

    if (lsMode) {
        std::vector<Connection> connections;
        GetTcpConnections(connections, f);
        GetUdpConnections(connections, f);
        std::sort(connections.begin(), connections.end(), [](const Connection& a, const Connection& b) {
            if (a.processName != b.processName) return a.processName < b.processName;
            return a.pid < b.pid;
        });
        if (f.unique) {
            auto last = std::unique(connections.begin(), connections.end(), [](const Connection& a, const Connection& b) {
                return a.processName == b.processName;
            });
            connections.erase(last, connections.end());
        }
        PrintTable(connections, -1, f.unique);
    } else {
        RunTui(f);
    }

    WSACleanup();
    return 0;
}
