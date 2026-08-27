package transport

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

// RESP2 最小协议读写，只覆盖本项目需要的几种类型。
const (
	respPrefixString  = '+'
	respPrefixError   = '-'
	respPrefixInteger = ':'
	respPrefixBulk    = '$'
	respPrefixArray   = '*'
)

// RESPReader 读取 RESP2 消息。
// ReadMessage 返回内容值，不包含类型前缀；数组会递归扁平化。
type RESPReader struct {
	r *bufio.Reader
}

// NewRESPReader 创建 RESP 读取器。
func NewRESPReader(r *bufio.Reader) *RESPReader {
	return &RESPReader{r: r}
}

// ReadMessage 读取一条 RESP 消息。
// 简单类型返回长度为 1 的切片；数组递归读取后扁平返回。
func (rd *RESPReader) ReadMessage() ([]string, error) {
	line, err := rd.r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if len(line) < 2 || line[len(line)-2] != '\r' {
		return nil, fmt.Errorf("invalid RESP line ending")
	}
	prefix := line[0]
	payload := line[1 : len(line)-2]

	switch prefix {
	case respPrefixString, respPrefixError, respPrefixInteger:
		return []string{payload}, nil
	case respPrefixBulk:
		return rd.readBulk(payload)
	case respPrefixArray:
		return rd.readArray(payload)
	default:
		return nil, fmt.Errorf("unsupported RESP prefix: %c", prefix)
	}
}

func (rd *RESPReader) readBulk(payload string) ([]string, error) {
	if payload == "-1" || payload == "" {
		return []string{""}, nil
	}
	n, err := strconv.Atoi(payload)
	if err != nil {
		return nil, fmt.Errorf("invalid bulk length: %s", payload)
	}

	buf := make([]byte, n+2)
	_, err = io.ReadFull(rd.r, buf)
	if err != nil {
		return nil, err
	}
	return []string{string(buf[:n])}, nil
}

func (rd *RESPReader) readArray(payload string) ([]string, error) {
	if payload == "-1" || payload == "" {
		return []string{}, nil
	}
	n, err := strconv.Atoi(payload)
	if err != nil {
		return nil, fmt.Errorf("invalid array length: %s", payload)
	}

	result := make([]string, 0, n)
	for i := 0; i < n; i++ {
		elems, err := rd.ReadMessage()
		if err != nil {
			return nil, err
		}
		result = append(result, elems...)
	}
	return result, nil
}

// RESPWriter 写入 RESP2 响应。
type RESPWriter struct {
	w *bufio.Writer
}

// NewRESPWriter 创建 RESP 写入器。
func NewRESPWriter(w *bufio.Writer) *RESPWriter {
	return &RESPWriter{w: w}
}

// WriteOK 写入 +OK。
func (w *RESPWriter) WriteOK() error {
	return w.writeLine("+OK")
}

// WriteError 写入 -ERR message。
func (w *RESPWriter) WriteError(message string) error {
	return w.writeLine("-ERR " + message)
}

// WriteInteger 写入 :N。
func (w *RESPWriter) WriteInteger(n int) error {
	return w.writeLine(fmt.Sprintf(":%d", n))
}

// WriteBulk 写入 $len\r\ndata\r\n。
func (w *RESPWriter) WriteBulk(data string) error {
	if err := w.writeLine(fmt.Sprintf("$%d", len(data))); err != nil {
		return err
	}
	if err := w.writeLine(data); err != nil {
		return err
	}
	return nil
}

// WriteNullBulk 写入 $-1。
func (w *RESPWriter) WriteNullBulk() error {
	return w.writeLine("$-1")
}

// WriteArray 写入 *N\r\n 后跟 N 条消息。
func (w *RESPWriter) WriteArray(count int, writer func(*RESPWriter) error) error {
	if err := w.writeLine(fmt.Sprintf("*%d", count)); err != nil {
		return err
	}
	if writer == nil {
		return nil
	}
	return writer(w)
}

// Flush 刷新底层 writer。
func (w *RESPWriter) Flush() error {
	return w.w.Flush()
}

func (w *RESPWriter) writeLine(line string) error {
	_, err := fmt.Fprintf(w.w, "%s\r\n", line)
	return err
}
