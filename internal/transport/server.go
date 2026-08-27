package transport

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"kv-cache/internal/command"
)

// Server 是 TCP 网络服务，使用 Executor 执行命令。
type Server struct {
	executor *command.Executor
	ln       net.Listener
	addr     string
}

// NewServer 创建网络服务实例。
func NewServer(executor *command.Executor, addr string) *Server {
	return &Server{
		executor: executor,
		addr:    addr,
	}
}

// Start 启动 TCP 监听并在后台接受连接。
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.addr, err)
	}
	s.ln = ln
	log.Printf("* Server listening on %s", s.addr)

	go s.acceptLoop()
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	rd := NewRESPReader(bufioReader(conn))
	wr := NewRESPWriter(bufioWriter(conn))

	for {
		elems, err := rd.ReadMessage()
		if err != nil {
			return
		}

		parts, err := parseRESPArray(elems)
		if err != nil {
			_ = wr.WriteError(err.Error())
			_ = wr.Flush()
			continue
		}

		if len(parts) == 0 {
			_ = wr.WriteError("empty command")
			_ = wr.Flush()
			continue
		}

		cmd := strings.ToUpper(parts[0])

		if cmd == "QUIT" || cmd == "EXIT" {
			_ = wr.WriteOK()
			_ = wr.Flush()
			return
		}

		result, err := s.executor.Execute(parts)
		if err != nil {
			_ = wr.WriteError(err.Error())
			_ = wr.Flush()
			continue
		}

		if result.Quit {
			_ = wr.WriteOK()
			_ = wr.Flush()
			return
		}

		_ = writeResult(wr, result)
		_ = wr.Flush()
	}
}

// writeResult 将 executor 的 Result 编码为 RESP 响应。
func writeResult(wr *RESPWriter, result *command.Result) error {
	if len(result.Lines) == 0 {
		return wr.WriteOK()
	}

	lines := result.Lines
	if len(lines) == 1 {
		line := lines[0]
		if line == "OK" {
			return wr.WriteOK()
		}
		if strings.HasPrefix(line, "(nil)") {
			return wr.WriteNullBulk()
		}
		if strings.HasPrefix(line, ":") || strings.HasPrefix(line, "(integer)") {
			val := strings.TrimPrefix(line, "(integer)")
			val = strings.TrimSpace(val)
			n, _ := parseInt(val)
			return wr.WriteInteger(n)
		}
		if strings.HasPrefix(line, "(empty array)") {
			return wr.WriteArray(0, nil)
		}
		// 默认作为 bulk string 返回
		return wr.WriteBulk(line)
	}

	// 多行作为数组返回
	err := wr.WriteArray(len(lines), func(w *RESPWriter) error {
		for _, line := range lines {
			if line == "OK" {
				if err := w.WriteOK(); err != nil {
					return err
				}
			} else {
				if err := w.WriteBulk(line); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return err
}

func parseInt(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// Close 关闭监听。
func (s *Server) Close() error {
	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}

func parseRESPArray(elems []string) ([]string, error) {
	if len(elems) == 0 {
		return nil, fmt.Errorf("empty RESP message")
	}
	return elems, nil
}

func bufioReader(conn net.Conn) *bufio.Reader {
	return bufio.NewReader(conn)
}

func bufioWriter(conn net.Conn) *bufio.Writer {
	return bufio.NewWriter(conn)
}
