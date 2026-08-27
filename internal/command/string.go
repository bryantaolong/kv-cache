package command

import (
	"fmt"
	"strconv"
	"time"
)

func (e *Executor) handleSet(parts []string) (*Result, error) {
	if len(parts) < 3 {
		return nil, fmt.Errorf("wrong number of arguments for 'set' command")
	}
	key, value := parts[1], parts[2]
	ttl := time.Duration(0)
	if len(parts) >= 4 {
		sec, err := strconv.Atoi(parts[3])
		if err != nil {
			return nil, fmt.Errorf("invalid TTL")
		}
		ttl = time.Duration(sec) * time.Second
	}
	if err := e.store.SetString(key, value, ttl); err != nil {
		return nil, err
	}
	if err := e.appendPersist(fmt.Sprintf("SET %s %s", key, value)); err != nil {
		return nil, err
	}
	return &Result{Lines: []string{"OK"}}, nil
}

func (e *Executor) handleGet(parts []string) (*Result, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'get' command")
	}
	val, ok := e.store.GetString(parts[1])
	if !ok {
		return &Result{Lines: []string{"(nil)"}}, nil
	}
	return &Result{Lines: []string{val}}, nil
}
