package proxy

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"http-proxy-platform/internal/config"
)

const (
	socksVer5             = 0x05
	socksMethodNoAuth     = 0x00
	socksMethodUserPass   = 0x02
	socksMethodNoAccept   = 0xff
	socksUserPassVersion  = 0x01
	socksCmdConnect       = 0x01
	socksAddrTypeIPv4     = 0x01
	socksAddrTypeDomain   = 0x03
	socksAddrTypeIPv6     = 0x04
	socksReplySucceeded   = 0x00
	socksReplyGeneralFail = 0x01
	socksReplyCmdNotSup   = 0x07
)

type SOCKS5Server struct {
	cfg      config.Config
	auth     Authenticator
	usage    UsageRecorder
	listener net.Listener
}

func NewSOCKS5Server(cfg config.Config, auth Authenticator, usage UsageRecorder) *SOCKS5Server {
	return &SOCKS5Server{cfg: cfg, auth: auth, usage: usage}
}

func (s *SOCKS5Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.SOCKS5ListenAddr())
	if err != nil {
		return err
	}
	s.listener = ln

	go func() {
		<-ctx.Done()
		_ = s.listener.Close()
	}()

	var wg sync.WaitGroup
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			log.Printf("socks5 accept error: %v", err)
			continue
		}
		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			s.handleConn(c)
		}(conn)
	}
	wg.Wait()
	return nil
}

func (s *SOCKS5Server) handleConn(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	sourceIP := sourceIPFromAddr(conn.RemoteAddr().String())

	username, err := s.handshake(conn, sourceIP)
	if err != nil {
		return
	}
	if err := s.handleConnect(conn, username); err != nil {
		return
	}
}

func (s *SOCKS5Server) handshake(conn net.Conn, sourceIP string) (string, error) {
	head := make([]byte, 2)
	if _, err := io.ReadFull(conn, head); err != nil {
		return "", err
	}
	if head[0] != socksVer5 {
		return "", errors.New("invalid socks version")
	}
	nMethods := int(head[1])
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return "", err
	}

	needAuth := true
	if static, ok := s.auth.(*StaticAuthenticator); ok {
		needAuth = static.Enabled()
	}

	selected := byte(socksMethodNoAccept)
	for _, m := range methods {
		if needAuth && m == socksMethodUserPass {
			selected = socksMethodUserPass
			break
		}
		if !needAuth && m == socksMethodNoAuth {
			selected = socksMethodNoAuth
			break
		}
	}
	if _, err := conn.Write([]byte{socksVer5, selected}); err != nil {
		return "", err
	}
	if selected == socksMethodNoAccept {
		return "", errors.New("no acceptable auth method")
	}
	if selected == socksMethodNoAuth {
		return "", nil
	}
	return s.handleUserPassAuth(conn, sourceIP)
}

func (s *SOCKS5Server) handleUserPassAuth(conn net.Conn, sourceIP string) (string, error) {
	head := make([]byte, 2)
	if _, err := io.ReadFull(conn, head); err != nil {
		return "", err
	}
	if head[0] != socksUserPassVersion {
		return "", errors.New("invalid auth version")
	}

	uLen := int(head[1])
	uname := make([]byte, uLen)
	if _, err := io.ReadFull(conn, uname); err != nil {
		return "", err
	}

	pLenBuf := make([]byte, 1)
	if _, err := io.ReadFull(conn, pLenBuf); err != nil {
		return "", err
	}
	pLen := int(pLenBuf[0])
	passwd := make([]byte, pLen)
	if _, err := io.ReadFull(conn, passwd); err != nil {
		return "", err
	}

	username := string(uname)
	if !s.auth.Validate(username, string(passwd), sourceIP) {
		_, _ = conn.Write([]byte{socksUserPassVersion, 0x01})
		return "", errors.New("authentication failed")
	}
	_, _ = conn.Write([]byte{socksUserPassVersion, 0x00})
	return username, nil
}

func (s *SOCKS5Server) handleConnect(conn net.Conn, username string) error {
	req := make([]byte, 4)
	if _, err := io.ReadFull(conn, req); err != nil {
		return err
	}
	if req[0] != socksVer5 {
		return errors.New("invalid request version")
	}
	if req[1] != socksCmdConnect {
		s.writeReply(conn, socksReplyCmdNotSup)
		return errors.New("unsupported command")
	}

	target, err := readTargetAddress(conn, req[3])
	if err != nil {
		s.writeReply(conn, socksReplyGeneralFail)
		return err
	}

	remote, err := net.DialTimeout("tcp", target, s.cfg.DialTimeout)
	if err != nil {
		s.writeReply(conn, socksReplyGeneralFail)
		return err
	}
	defer remote.Close()

	if err := s.writeReply(conn, socksReplySucceeded); err != nil {
		return err
	}

	_ = conn.SetDeadline(time.Time{})
	tunnelWithUsage(conn, remote, func(total int64) {
		s.recordUsage(username, total)
	})
	return nil
}

func (s *SOCKS5Server) writeReply(conn net.Conn, code byte) error {
	reply := []byte{socksVer5, code, 0x00, socksAddrTypeIPv4, 0, 0, 0, 0, 0, 0}
	_, err := conn.Write(reply)
	return err
}

func readTargetAddress(conn net.Conn, addrType byte) (string, error) {
	var host string
	switch addrType {
	case socksAddrTypeIPv4:
		buf := make([]byte, 4)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return "", err
		}
		host = net.IP(buf).String()
	case socksAddrTypeIPv6:
		buf := make([]byte, 16)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return "", err
		}
		host = net.IP(buf).String()
	case socksAddrTypeDomain:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return "", err
		}
		domain := make([]byte, int(lenBuf[0]))
		if _, err := io.ReadFull(conn, domain); err != nil {
			return "", err
		}
		host = string(domain)
	default:
		return "", errors.New("unsupported address type")
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return "", err
	}
	port := binary.BigEndian.Uint16(portBuf)

	return net.JoinHostPort(host, strconv.Itoa(int(port))), nil
}

func (s *SOCKS5Server) recordUsage(username string, bytes int64) {
	if username == "" || bytes <= 0 {
		return
	}
	if s.usage == nil {
		return
	}
	s.usage.RecordUsage(username, bytes)
}
