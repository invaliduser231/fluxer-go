package fluxer

import "strings"

var routeWords = map[string]bool{"@me": true, "bot": true, "bulk-delete": true}

func MetricRoute(route string) string {
	method, path, _ := strings.Cut(route, " ")
	parts := strings.Split(path, "/")
	if len(parts) > 9 {
		parts = append(parts[:9], "…")
	}
	for i, p := range parts {
		switch {
		case i == 0 || p == "…":
		case i%2 == 1 && !isWord(p):
			parts[i] = ":?"
		case i%2 == 0 && !routeWords[p]:
			parts[i] = ":id"
		}
	}
	return method + " " + strings.Join(parts, "/")
}

func isWord(s string) bool {
	if s == "" || len(s) > 24 {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && r != '-' && r != '@' {
			return false
		}
	}
	return true
}
