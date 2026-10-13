package web

import (
	"net"
	"net/http"
	"net/url"
)

// LocalOriginCheck is a websocket.Upgrader CheckOrigin that admits only pages served by the user's
// own machine: the request's own host, a loopback host (localhost, 127.0.0.1, ::1 -- compared
// exactly, so "localhost.evil.com" does not pass), or the Prism GUI's Wails webview (the wails://
// scheme on macOS/Linux, wails.localhost on Windows). A request with no Origin is a native client,
// not a browser, and is admitted.
//
// Browsers send Origin on every WebSocket handshake and do not apply the same-origin policy to it,
// so without this check any web page the user visits could open these sockets -- the GUI terminal
// server, for one, hands out an SSH session to the user's instances (cross-site WebSocket
// hijacking).
func LocalOriginCheck(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if u.Scheme == "wails" {
		return true
	}
	host := hostOnly(u.Host)
	switch host {
	case "localhost", "127.0.0.1", "::1", "wails.localhost":
		return true
	}
	return host != "" && host == hostOnly(r.Host)
}

// hostOnly strips any :port (and IPv6 brackets) from a host[:port] string.
func hostOnly(hostport string) string {
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		return h
	}
	return hostport
}
