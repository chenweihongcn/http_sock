package netutil

import (
	"net"
	"net/http"
	"strings"
)

type ClientIPInfo struct {
	ClientIP     string
	RemoteIP     string
	ForwardedFor string
	RealIPHeader string
}

func ResolveClientIP(r *http.Request, trustProxyHeaders bool, realIPHeader string) ClientIPInfo {
	headerName := strings.TrimSpace(realIPHeader)
	if headerName == "" {
		headerName = "X-Forwarded-For"
	}

	remote := NormalizeIP(FromRemoteAddr(r.RemoteAddr))
	info := ClientIPInfo{
		ClientIP:     remote,
		RemoteIP:     remote,
		ForwardedFor: strings.TrimSpace(r.Header.Get("X-Forwarded-For")),
		RealIPHeader: headerName,
	}
	if !trustProxyHeaders {
		return info
	}

	candidates := make([]string, 0, 3)
	if preferred := strings.TrimSpace(r.Header.Get(headerName)); preferred != "" {
		candidates = append(candidates, preferred)
	}
	if !strings.EqualFold(headerName, "X-Forwarded-For") {
		if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
			candidates = append(candidates, forwarded)
		}
	}
	if !strings.EqualFold(headerName, "X-Real-IP") {
		if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
			candidates = append(candidates, realIP)
		}
	}

	for _, raw := range candidates {
		if ip := firstValidIP(raw); ip != "" {
			info.ClientIP = ip
			break
		}
	}
	return info
}

func FromRemoteAddr(addr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return strings.TrimSpace(addr)
	}
	return strings.TrimSpace(host)
}

func NormalizeIP(raw string) string {
	if ip := parseIP(raw); ip != nil {
		if v4 := ip.To4(); v4 != nil {
			return v4.String()
		}
		return ip.String()
	}
	return strings.TrimSpace(raw)
}

func firstValidIP(raw string) string {
	for _, part := range strings.Split(raw, ",") {
		if ip := NormalizeIP(part); parseIP(ip) != nil {
			return ip
		}
	}
	return ""
}

func parseIP(raw string) net.IP {
	raw = strings.TrimSpace(strings.Trim(raw, "\"'"))
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, "[") && strings.Contains(raw, "]") {
		raw = raw[1:strings.Index(raw, "]")]
	}
	if ip := net.ParseIP(raw); ip != nil {
		return ip
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		return net.ParseIP(strings.TrimSpace(host))
	}
	return nil
}
