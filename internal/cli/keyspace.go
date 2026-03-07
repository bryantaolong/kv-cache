package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (c *CLI) handleDel(parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("wrong number of arguments for 'del' command")
	}
	count := 0
	for i := 1; i < len(parts); i++ {
		if c.store.Delete(parts[i]) {
			count++
		}
	}
	fmt.Fprintf(c.writer, "(integer) %d\n", count)
	return nil
}

func (c *CLI) handleKeys(parts []string) error {
	pattern := "*"
	if len(parts) >= 2 {
		pattern = parts[1]
	}

	keys := c.store.Keys()
	if len(keys) == 0 {
		fmt.Fprintln(c.writer, "(empty array)")
		return nil
	}

	matched := make([]string, 0)
	for _, k := range keys {
		if matchPattern(k, pattern) {
			matched = append(matched, k)
		}
	}

	if len(matched) == 0 {
		fmt.Fprintln(c.writer, "(empty array)")
		return nil
	}
	for i, k := range matched {
		fmt.Fprintf(c.writer, "%d) \"%s\"\n", i+1, k)
	}
	return nil
}

func matchPattern(s, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return s == pattern
	}
	parts := strings.Split(pattern, "*")
	if len(parts) == 2 {
		if parts[0] == "" {
			return strings.HasSuffix(s, parts[1])
		}
		if parts[1] == "" {
			return strings.HasPrefix(s, parts[0])
		}
		return strings.HasPrefix(s, parts[0]) && strings.HasSuffix(s, parts[1])
	}
	return false
}

func (c *CLI) handleFlushDB() error {
	c.store.Flush()
	fmt.Fprintln(c.writer, "OK")
	return nil
}

func (c *CLI) handleExpire(parts []string) error {
	if len(parts) < 3 {
		return fmt.Errorf("wrong number of arguments for 'expire' command")
	}
	sec, err := strconv.Atoi(parts[2])
	if err != nil {
		return fmt.Errorf("invalid TTL")
	}
	if c.store.Expire(parts[1], time.Duration(sec)*time.Second) {
		fmt.Fprintln(c.writer, "(integer) 1")
	} else {
		fmt.Fprintln(c.writer, "(integer) 0")
	}
	return nil
}

func (c *CLI) handleTTL(parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("wrong number of arguments for 'ttl' command")
	}
	ttl := c.store.TTL(parts[1])
	fmt.Fprintf(c.writer, "(integer) %d\n", int(ttl.Seconds()))
	return nil
}
