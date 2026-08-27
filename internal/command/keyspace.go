package command

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (e *Executor) handleDel(parts []string) (*Result, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'del' command")
	}
	count := 0
	for i := 1; i < len(parts); i++ {
		if e.store.Delete(parts[i]) {
			count++
		}
	}
	if err := e.appendPersist("DEL " + strings.Join(parts[1:], " ")); err != nil {
		return nil, err
	}
	return &Result{Lines: []string{fmt.Sprintf("(integer) %d", count)}}, nil
}

func (e *Executor) handleKeys(parts []string) (*Result, error) {
	pattern := "*"
	if len(parts) >= 2 {
		pattern = parts[1]
	}

	keys := e.store.Keys()
	matched := make([]string, 0)
	for _, k := range keys {
		if matchPattern(k, pattern) {
			matched = append(matched, k)
		}
	}

	if len(matched) == 0 {
		return &Result{Lines: []string{"(empty array)"}}, nil
	}
	lines := make([]string, 0, len(matched))
	for i, k := range matched {
		lines = append(lines, fmt.Sprintf("%d) \"%s\"", i+1, k))
	}
	return &Result{Lines: lines}, nil
}

func (e *Executor) handleFlushDB() (*Result, error) {
	e.store.Flush()
	if err := e.appendPersist("FLUSHDB"); err != nil {
		return nil, err
	}
	return &Result{Lines: []string{"OK"}}, nil
}

func (e *Executor) handleExpire(parts []string) (*Result, error) {
	if len(parts) < 3 {
		return nil, fmt.Errorf("wrong number of arguments for 'expire' command")
	}
	sec, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid TTL")
	}
	ok := e.store.Expire(parts[1], time.Duration(sec)*time.Second)
	if err := e.appendPersist(fmt.Sprintf("EXPIRE %s %s", parts[1], parts[2])); err != nil {
		return nil, err
	}
	if ok {
		return &Result{Lines: []string{":1"}}, nil
	}
	return &Result{Lines: []string{":0"}}, nil
}

func (e *Executor) handleTTL(parts []string) (*Result, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'ttl' command")
	}
	ttl := e.store.TTL(parts[1])
	return &Result{Lines: []string{fmt.Sprintf(":%d", int(ttl.Seconds()))}}, nil
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
