package transport

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
)

// Client 是网络客户端，通过 RESP2 与 server 通信。
type Client struct {
	conn net.Conn
	rd   *RESPReader
	wr   *RESPWriter
}

// NewClient 连接到 server。
func NewClient(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	return &Client{
		conn: conn,
		rd:   NewRESPReader(bufio.NewReader(conn)),
		wr:   NewRESPWriter(bufio.NewWriter(conn)),
	}, nil
}

// Execute 发送一条命令并读取响应。
func (c *Client) Execute(args []string) ([]string, error) {
	if err := c.writeArray(args); err != nil {
		return nil, err
	}
	if err := c.wr.Flush(); err != nil {
		return nil, err
	}

	return c.readResponse()
}

// Close 关闭连接。
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) writeArray(args []string) error {
	if err := c.wr.WriteArray(len(args), nil); err != nil {
		return err
	}
	for _, arg := range args {
		if err := c.wr.WriteBulk(arg); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) readResponse() ([]string, error) {
	line, err := c.rd.r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if len(line) < 2 || line[len(line)-2] != '\r' {
		return nil, fmt.Errorf("invalid RESP line ending")
	}

	prefix := line[0]
	payload := line[1 : len(line)-2]

	switch prefix {
	case respPrefixString:
		return []string{payload}, nil
	case respPrefixError:
		return []string{"(error) " + payload}, nil
	case respPrefixInteger:
		return []string{":" + payload}, nil
	case respPrefixBulk:
		n, err := parseLength(payload)
		if err != nil || n < 0 {
			return []string{"(nil)"}, nil
		}
		buf := make([]byte, n+2)
		_, err = io.ReadFull(c.rd.r, buf)
		if err != nil {
			return nil, err
		}
		return []string{string(buf[:n])}, nil
	case respPrefixArray:
		n, err := parseLength(payload)
		if err != nil || n <= 0 {
			return []string{"(empty array)"}, nil
		}
		var results []string
		for i := 0; i < n; i++ {
			elems, err := c.readResponse()
			if err != nil {
				return nil, err
			}
			results = append(results, elems...)
		}
		return results, nil
	default:
		return nil, fmt.Errorf("unsupported RESP prefix: %c", prefix)
	}
}

func parseLength(s string) (int, error) {
	if s == "-1" {
		return -1, nil
	}
	return strconv.Atoi(s)
}
