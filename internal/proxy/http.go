package proxy

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"http-proxy-platform/internal/config"
	"http-proxy-platform/internal/netutil"
)

type HTTPProxyServer struct {
	cfg    config.Config
	auth   Authenticator
	usage  UsageRecorder
	srv    *http.Server
}

func NewHTTPProxyServer(cfg config.Config, auth Authenticator, usage UsageRecorder) *HTTPProxyServer {
	h := &HTTPProxyServer{cfg: cfg, auth: auth, usage: usage}
	h.srv = &http.Server{
		Addr:         cfg.HTTPListenAddr(),
		Handler:      h,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return h
}

func (h *HTTPProxyServer) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("http shutdown error: %v", err)
		}
	}()

	go func() {
		if err := h.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	return <-errCh
}

func (h *HTTPProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	username, ok := h.authorizeHTTP(r)
	if !ok {
		w.Header().Set("Proxy-Authenticate", `Basic realm="proxy"`)
		http.Error(w, "proxy auth required", http.StatusProxyAuthRequired)
		return
	}

	if r.Method == http.MethodConnect {
		h.handleConnect(w, r, username)
		return
	}
	h.handleForward(w, r, username)
}

func (h *HTTPProxyServer) authorizeHTTP(r *http.Request) (string, bool) {
	pAuth := r.Header.Get("Proxy-Authorization")
	if pAuth == "" {
		return "", false
	}
	parts := strings.SplitN(pAuth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Basic") {
		return "", false
	}
	raw, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}
	pair := strings.SplitN(string(raw), ":", 2)
	if len(pair) != 2 {
		return "", false
	}
	ipInfo := netutil.ResolveClientIP(r, h.cfg.TrustProxyHeaders, h.cfg.RealIPHeader)
	sourceIP := ipInfo.ClientIP
	clientAgent := strings.TrimSpace(r.UserAgent())
	if clientAgent == "" {
		if r.Method == http.MethodConnect {
			clientAgent = "HTTP CONNECT 客户端"
		} else {
			clientAgent = "HTTP 代理客户端"
		}
	}
	if !h.auth.Validate(pair[0], pair[1], sourceIP, clientAgent) {
		return "", false
	}
	return pair[0], true
}

func (h *HTTPProxyServer) handleConnect(w http.ResponseWriter, r *http.Request, username string) {
	targetConn, err := net.DialTimeout("tcp", r.Host, h.cfg.DialTimeout)
	if err != nil {
		http.Error(w, "dial target failed", http.StatusServiceUnavailable)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		targetConn.Close()
		http.Error(w, "hijack not supported", http.StatusInternalServerError)
		return
	}

	clientConn, rw, err := hijacker.Hijack()
	if err != nil {
		targetConn.Close()
		http.Error(w, "hijack failed", http.StatusInternalServerError)
		return
	}

	_, _ = rw.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
	_ = rw.Flush()

	tunnelWithUsage(clientConn, targetConn, func(total int64) {
		h.recordUsage(username, total)
	}, h.speedLimitBPS(username))
}

func (h *HTTPProxyServer) handleForward(w http.ResponseWriter, r *http.Request, username string) {
	transport := &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{Timeout: h.cfg.DialTimeout}).DialContext,
	}

	outReq := new(http.Request)
	*outReq = *r
	outReq.RequestURI = ""
	outReq.Header = r.Header.Clone()
	outReq.Header.Del("Proxy-Authorization")

	resp, err := transport.RoundTrip(outReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("forward failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	n, _ := copyWithLimit(w, resp.Body, h.speedLimitBPS(username))
	h.recordUsage(username, n)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func tunnelWithUsage(a, b net.Conn, onDone func(total int64), bytesPerSec int64) {
	defer a.Close()
	defer b.Close()

	done := make(chan int64, 2)
	go func() {
		n, _ := copyWithLimit(a, b, bytesPerSec)
		done <- n
	}()
	go func() {
		n, _ := copyWithLimit(b, a, bytesPerSec)
		done <- n
	}()
	total := <-done
	total += <-done
	if onDone != nil {
		onDone(total)
	}
}

func sourceIPFromAddr(addr string) string {
	return netutil.NormalizeIP(netutil.FromRemoteAddr(addr))
}

func (h *HTTPProxyServer) recordUsage(username string, bytes int64) {
	if username == "" || bytes <= 0 {
		return
	}
	if h.usage == nil {
		return
	}
	h.usage.RecordUsage(username, bytes)
}

func (h *HTTPProxyServer) speedLimitBPS(username string) int64 {
	if h.usage == nil || username == "" {
		return 0
	}
	return h.usage.SpeedLimitBytesPerSec(username)
}

func copyWithLimit(dst io.Writer, src io.Reader, bytesPerSec int64) (int64, error) {
	if bytesPerSec <= 0 {
		return io.Copy(dst, src)
	}
	buf := make([]byte, 16*1024)
	var total int64
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			total += int64(nw)
			if ew != nil {
				return total, ew
			}
			if nw != nr {
				return total, io.ErrShortWrite
			}
			sleep := time.Duration((float64(nr) / float64(bytesPerSec)) * float64(time.Second))
			if sleep > 0 {
				time.Sleep(sleep)
			}
		}
		if er != nil {
			if errors.Is(er, io.EOF) {
				return total, nil
			}
			return total, er
		}
	}
}
