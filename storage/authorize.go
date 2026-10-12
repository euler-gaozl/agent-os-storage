package storage

import "strings"

func Authorize(caller, method, path string) bool {
	if path == "/readyz" || path == "/metrics" {
		return method == "GET" && (caller == "storage" || caller == "control")
	}
	if caller == "runtime" || caller == "worker" {
		return method == "PUT" && strings.HasPrefix(path, "/v1/artifacts/") || caller == "runtime" && method == "POST" && path == "/v1/artifacts/page"
	}
	return caller == "control" && (method == "GET" && strings.HasPrefix(path, "/artifacts/") || method == "POST" && path == "/gc")
}
