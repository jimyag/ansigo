package facts

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/jimyag/ansigo/pkg/connection"
)

// Facts represents ansible facts
type Facts map[string]interface{}

// GatherFacts collects system information from the remote host
func GatherFacts(conn *connection.Connection) (Facts, error) {
	facts := make(Facts)

	// Gather basic system facts
	if err := gatherSystemFacts(conn, facts); err != nil {
		return nil, fmt.Errorf("failed to gather system facts: %w", err)
	}

	// Gather architecture facts
	if err := gatherArchitectureFacts(conn, facts); err != nil {
		return nil, fmt.Errorf("failed to gather architecture facts: %w", err)
	}

	// Gather distribution facts (Linux only)
	if facts["ansible_system"] == "Linux" {
		if err := gatherDistributionFacts(conn, facts); err != nil {
			// Non-fatal - just log and continue
			// Some systems may not have standard release files
		}
	}

	return facts, nil
}

// gatherSystemFacts gathers OS type information
func gatherSystemFacts(conn *connection.Connection, facts Facts) error {
	// Get OS type using uname -s
	output, err := conn.ExecuteCommand("uname -s")
	if err != nil {
		// Fallback to Go runtime
		facts["ansible_system"] = runtime.GOOS
	} else {
		system := strings.TrimSpace(string(output))
		facts["ansible_system"] = system
	}

	return nil
}

// gatherArchitectureFacts gathers CPU architecture information
func gatherArchitectureFacts(conn *connection.Connection, facts Facts) error {
	// Get architecture using uname -m
	output, err := conn.ExecuteCommand("uname -m")
	if err != nil {
		// Fallback to Go runtime
		facts["ansible_architecture"] = runtime.GOARCH
	} else {
		arch := strings.TrimSpace(string(output))
		facts["ansible_architecture"] = arch
	}

	return nil
}

// gatherDistributionFacts gathers Linux distribution information
func gatherDistributionFacts(conn *connection.Connection, facts Facts) error {
	// Try to get distribution from /etc/os-release (modern Linux)
	output, err := conn.ExecuteCommand("cat /etc/os-release")
	if err == nil {
		parseOSRelease(string(output), facts)
		return nil
	}

	// Fallback: try /etc/lsb-release (Ubuntu/Debian)
	output, err = conn.ExecuteCommand("cat /etc/lsb-release")
	if err == nil {
		parseLSBRelease(string(output), facts)
		return nil
	}

	// Fallback: try /etc/redhat-release (RedHat/CentOS)
	output, err = conn.ExecuteCommand("cat /etc/redhat-release")
	if err == nil {
		parseRedHatRelease(string(output), facts)
		return nil
	}

	// No distribution info available
	return fmt.Errorf("could not detect distribution")
}

// parseOSRelease parses /etc/os-release format
func parseOSRelease(content string, facts Facts) {
	lines := strings.Split(content, "\n")
	osInfo := make(map[string]string)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		osInfo[key] = value
	}

	// Set ansible_distribution
	if id, ok := osInfo["ID"]; ok {
		// Capitalize first letter to match Ansible convention
		facts["ansible_distribution"] = strings.Title(id)
	} else if name, ok := osInfo["NAME"]; ok {
		facts["ansible_distribution"] = name
	}

	// Set ansible_distribution_version
	if version, ok := osInfo["VERSION_ID"]; ok {
		facts["ansible_distribution_version"] = version
	}

	// Set ansible_distribution_major_version
	if version, ok := facts["ansible_distribution_version"].(string); ok {
		majorVersion := strings.Split(version, ".")[0]
		facts["ansible_distribution_major_version"] = majorVersion
	}

	// Determine ansible_os_family
	if idLike, ok := osInfo["ID_LIKE"]; ok {
		facts["ansible_os_family"] = determineOSFamily(idLike)
	} else if id, ok := osInfo["ID"]; ok {
		facts["ansible_os_family"] = determineOSFamily(id)
	}
}

// parseLSBRelease parses /etc/lsb-release format
func parseLSBRelease(content string, facts Facts) {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")

		switch key {
		case "DISTRIB_ID":
			facts["ansible_distribution"] = value
		case "DISTRIB_RELEASE":
			facts["ansible_distribution_version"] = value
			majorVersion := strings.Split(value, ".")[0]
			facts["ansible_distribution_major_version"] = majorVersion
		}
	}

	// Determine OS family
	if dist, ok := facts["ansible_distribution"].(string); ok {
		facts["ansible_os_family"] = determineOSFamily(strings.ToLower(dist))
	}
}

// parseRedHatRelease parses /etc/redhat-release format
func parseRedHatRelease(content string, facts Facts) {
	content = strings.TrimSpace(content)

	// Example: "CentOS Linux release 7.9.2009 (Core)"
	if strings.Contains(strings.ToLower(content), "centos") {
		facts["ansible_distribution"] = "CentOS"
		facts["ansible_os_family"] = "RedHat"
	} else if strings.Contains(strings.ToLower(content), "red hat") {
		facts["ansible_distribution"] = "RedHat"
		facts["ansible_os_family"] = "RedHat"
	} else if strings.Contains(strings.ToLower(content), "fedora") {
		facts["ansible_distribution"] = "Fedora"
		facts["ansible_os_family"] = "RedHat"
	}

	// Try to extract version
	words := strings.Fields(content)
	for i, word := range words {
		if (word == "release" || word == "Release") && i+1 < len(words) {
			version := words[i+1]
			facts["ansible_distribution_version"] = version
			majorVersion := strings.Split(version, ".")[0]
			facts["ansible_distribution_major_version"] = majorVersion
			break
		}
	}
}

// determineOSFamily maps distribution to OS family
func determineOSFamily(distID string) string {
	distID = strings.ToLower(distID)

	switch {
	case strings.Contains(distID, "debian"):
		return "Debian"
	case strings.Contains(distID, "ubuntu"):
		return "Debian"
	case strings.Contains(distID, "rhel"):
		return "RedHat"
	case strings.Contains(distID, "centos"):
		return "RedHat"
	case strings.Contains(distID, "fedora"):
		return "RedHat"
	case strings.Contains(distID, "red hat"):
		return "RedHat"
	case strings.Contains(distID, "arch"):
		return "Arch"
	case strings.Contains(distID, "alpine"):
		return "Alpine"
	case strings.Contains(distID, "suse"):
		return "Suse"
	default:
		return "Unknown"
	}
}
