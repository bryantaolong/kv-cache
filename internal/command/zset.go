package command

import (
	"fmt"
	"strconv"
)

func (e *Executor) handleZAdd(parts []string) (*Result, error) {
	if len(parts) < 4 {
		return nil, fmt.Errorf("wrong number of arguments for 'zadd' command")
	}
	score, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid score")
	}
	n, err := e.store.ZAdd(parts[1], score, parts[3])
	if err != nil {
		return nil, err
	}
	if err := e.appendPersist(fmt.Sprintf("ZADD %s %f %s", parts[1], score, parts[3])); err != nil {
		return nil, err
	}
	if !e.loading {
		return &Result{Lines: []string{fmt.Sprintf("(integer) %d", n)}}, nil
	}
	return &Result{Lines: []string{fmt.Sprintf(":%d", n)}}, nil
}

func (e *Executor) handleZRange(parts []string) (*Result, error) {
	if len(parts) < 4 {
		return nil, fmt.Errorf("wrong number of arguments for 'zrange' command")
	}
	start, err1 := strconv.Atoi(parts[2])
	stop, err2 := strconv.Atoi(parts[3])
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("invalid range")
	}
	members, err := e.store.ZRange(parts[1], start, stop)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return &Result{Lines: []string{"(empty array)"}}, nil
	}
	lines := make([]string, 0, len(members))
	for i, m := range members {
		lines = append(lines, fmt.Sprintf("%d) \"%s\"", i+1, m.Member))
		lines = append(lines, fmt.Sprintf("   score: %g", m.Score))
	}
	return &Result{Lines: lines}, nil
}

func (e *Executor) handleZCard(parts []string) (*Result, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'zcard' command")
	}
	n, err := e.store.ZCard(parts[1])
	if err != nil {
		return nil, err
	}
	return &Result{Lines: []string{fmt.Sprintf("(integer) %d", n)}}, nil
}
