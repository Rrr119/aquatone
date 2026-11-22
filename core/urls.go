package core

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

var (
	securePorts = []int{443, 832, 981, 1010, 1311, 2083, 2087, 2095, 2096, 4712,
		7000, 8172, 8243, 8333, 8443, 8834, 9443, 12443, 18091, 18092}
)

func HostAndPortToURL(host string, port int, protocol string) string {
	var url string
	if protocol != "" {
		url = fmt.Sprintf("%s://%s", protocol, host)
	} else if isSecurePort(port) {
		url = fmt.Sprintf("https://%s", host)
	} else {
		url = fmt.Sprintf("http://%s", host)
	}
	if isStandardPort(port) {
		url = fmt.Sprintf("%s/", url)
	} else {
		url = fmt.Sprintf("%s:%d/", url, port)
	}
	return url
}

func isSecurePort(port int) bool {
	for _, p := range securePorts {
		if p == port {
			return true
		}
	}
	return false
}

func isStandardPort(port int) bool {
	if port == 80 || port == 443 {
		return true
	}
	return false
}

// TargetKey returns a stable key used to deduplicate reconnaissance targets.
// The key deliberately does not replace the original target: reports should
// continue to show the URL as it was supplied by the user.
func TargetKey(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}

	u, err := url.Parse(target)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return strings.ToLower(strings.TrimSuffix(target, "."))
	}

	u.Scheme = strings.ToLower(u.Scheme)
	hostname := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	port := u.Port()
	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		port = ""
	}

	host := hostname
	if ip := net.ParseIP(hostname); ip != nil && strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	if port != "" {
		host += ":" + port
	}
	u.Host = host
	u.Fragment = ""
	if u.Path == "" {
		u.Path = "/"
	}

	return u.String()
}

// DeduplicateTargets removes duplicate targets while preserving input order.
func DeduplicateTargets(targets []string) []string {
	seen := make(map[string]struct{}, len(targets))
	unique := make([]string, 0, len(targets))

	for _, target := range targets {
		key := TargetKey(target)
		if key == "" {
			continue
		}
		if _, found := seen[key]; found {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, target)
	}

	return unique
}
