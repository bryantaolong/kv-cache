package command

import (
	"fmt"
	"strconv"
	"strings"
)

func (e *Executor) handleLPush(parts []string) (*Result, error) {
	if len(parts) < 3 {
		return nil, fmt.Errorf("wrong number of arguments for 'lpush' command")
	}
	length, err := e.store.LPush(parts[1], parts[2:]...)
	if err != nil {
		return nil, err
	}
	if err := e.appendPersist("LPUSH " + parts[1] + " " + strings.Join(parts[2:], " ")); err != nil {
		return nil, err
	}
	if !e.loading {
		return &Result{Lines: []string{fmt.Sprintf("(integer) %d", length)}}, nil
	}
	return &Result{Lines: []string{fmt.Sprintf(":%d", length)}}, nil
}

func (e *Executor) handleRPush(parts []string) (*Result, error) {
	if len(parts) < 3 {
		return nil, fmt.Errorf("wrong number of arguments for 'rpush' command")
	}
	length, err := e.store.RPush(parts[1], parts[2:]...)
	if err != nil {
		return nil, err
	}
	if err := e.appendPersist("RPUSH " + parts[1] + " " + strings.Join(parts[2:], " ")); err != nil {
		return nil, err
	}
	if !e.loading {
		return &Result{Lines: []string{fmt.Sprintf("(integer) %d", length)}}, nil
	}
	return &Result{Lines: []string{fmt.Sprintf(":%d", length)}}, nil
}

func (e *Executor) handleLPop(parts []string) (*Result, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'lpop' command")
	}
	val, _, err := e.store.LPop(parts[1])
	if err != nil {
		return nil, err
	}
	if val == "" {
		return &Result{Lines: []string{"(nil)"}}, nil
	}
	return &Result{Lines: []string{val}}, nil
}

func (e *Executor) handleRPop(parts []string) (*Result, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'rpop' command")
	}
	val, _, err := e.store.RPop(parts[1])
	if err != nil {
		return nil, err
	}
	if val == "" {
		return &Result{Lines: []string{"(nil)"}}, nil
	}
	return &Result{Lines: []string{val}}, nil
}

func (e *Executor) handleLRange(parts []string) (*Result, error) {
	if len(parts) < 4 {
		return nil, fmt.Errorf("wrong number of arguments for 'lrange' command")
	}
	start, err1 := strconv.Atoi(parts[2])
	stop, err2 := strconv.Atoi(parts[3])
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("invalid range")
	}
	vals, err := e.store.LRange(parts[1], start, stop)
	if err != nil {
		return nil, err
	}
	if len(vals) == 0 {
		return &Result{Lines: []string{"(empty array)"}}, nil
	}
	lines := make([]string, 0, len(vals))
	for i, v := range vals {
		lines = append(lines, fmt.Sprintf("%d) \"%s\"", i+1, v))
	}
	return &Result{Lines: lines}, nil
}

func (e *Executor) handleLLen(parts []string) (*Result, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'llen' command")
	}
	n, err := e.store.LLen(parts[1])
	if err != nil {
		return nil, err
	}
	return &Result{Lines: []string{fmt.Sprintf("(integer) %d", n)}}, nil
}
