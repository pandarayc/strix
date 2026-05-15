package tool

import "strings"

// dangerous patterns that are blocked in shell commands.
var dangerousPatterns = []string{
	"rm -rf /",
	"rm -rf --no-preserve-root /",
	"dd if=/dev/zero",
	"mkfs.",
	"> /dev/sda",
	"fork bomb",
	":(){ :|:& };:",
	"chmod 777 /",
	"chown -R root /",
}

// blocklisted commands that require explicit user approval.
var blocklistedCommands = []string{
	"sudo ",
	"su ",
	"passwd",
	"shutdown",
	"reboot",
	"halt",
	"poweroff",
	"iptables",
	"ufw ",
	"mount ",
	"umount ",
	"fdisk ",
	"parted ",
}

// IsShellSafe checks if a command contains dangerous patterns.
func IsShellSafe(cmd string) (bool, string) {
	lower := strings.ToLower(cmd)

	for _, pattern := range dangerousPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return false, "blocked dangerous pattern: " + pattern
		}
	}

	for _, blocked := range blocklistedCommands {
		if strings.HasPrefix(lower, blocked) || strings.Contains(lower, " "+blocked) || strings.Contains(lower, ";"+blocked) || strings.Contains(lower, "&&"+blocked) || strings.Contains(lower, "|"+blocked) {
			return false, "blocked command: " + blocked
		}
	}

	return true, ""
}
