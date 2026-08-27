package command

import (
	"fmt"
	"strings"
)

func (e *Executor) handleSAdd(parts []string) (*Result, error) {
	if len(parts) < 3 {
		return nil, fmt.Errorf("wrong number of arguments for 'sadd' command")
	}
	n, err := e.store.SAdd(parts[1], parts[2:]...)
	if err != nil {
		return nil, err
	}
	if err := e.appendPersist("SADD " + parts[1] + " " + strings.Join(parts[2:], " ")); err != nil {
		return nil, err
	}
	if !e.loading {
		return &Result{Lines: []string{fmt.Sprintf("(integer) %d", n)}}, nil
	}
	return &Result{Lines: []string{fmt.Sprintf(":%d", n)}}, nil
}

func (e *Executor) handleSMembers(parts []string) (*Result, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'smembers' command")
	}
	members, err := e.store.SMembers(parts[1])
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return &Result{Lines: []string{"(empty array)"}}, nil
	}
	lines := make([]string, 0, len(members))
	for i, m := range members {
		lines = append(lines, fmt.Sprintf("%d) \"%s\"", i+1, m))
	}
	return &Result{Lines: lines}, nil
}

func (e *Executor) handleSCard(parts []string) (*Result, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'scard' command")
	}
	n, err := e.store.SCard(parts[1])
	if err != nil {
		return nil, err
	}
	return &Result{Lines: []string{fmt.Sprintf("(integer) %d", n)}}, nil
}
