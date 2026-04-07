package requestip

import (
	"net"
	"net/http"
	"strings"
)

var clientIPHeaders = []string{
	"X-Forwarded-For",
	"X-Real-IP",
	"X-Client-IP",
	"CF-Connecting-IP",
}

func FromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	for i := range clientIPHeaders {
		value := normalizeIPValue(r.Header.Get(clientIPHeaders[i]))
		if value != "" {
			return value
		}
	}
	return normalizeIPValue(r.RemoteAddr)
}

func normalizeIPValue(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, ",") {
		parts := strings.Split(raw, ",")
		for i := range parts {
			part := strings.TrimSpace(parts[i])
			if part != "" && !strings.EqualFold(part, "unknown") {
				raw = part
				break
			}
		}
	}
	if strings.EqualFold(raw, "unknown") {
		return ""
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		return host
	}
	return raw
}
