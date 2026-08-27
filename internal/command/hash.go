package command

import "fmt"

func (e *Executor) handleHSet(parts []string) (*Result, error) {
	if len(parts) < 4 {
		return nil, fmt.Errorf("wrong number of arguments for 'hset' command")
	}
	n, err := e.store.HSet(parts[1], parts[2], parts[3])
	if err != nil {
		return nil, err
	}
	if err := e.appendPersist(fmt.Sprintf("HSET %s %s %s", parts[1], parts[2], parts[3])); err != nil {
		return nil, err
	}
	if !e.loading {
		return &Result{Lines: []string{fmt.Sprintf("(integer) %d", n)}}, nil
	}
	return &Result{Lines: []string{fmt.Sprintf(":%d", n)}}, nil
}

func (e *Executor) handleHGet(parts []string) (*Result, error) {
	if len(parts) < 3 {
		return nil, fmt.Errorf("wrong number of arguments for 'hget' command")
	}
	val, ok, err := e.store.HGet(parts[1], parts[2])
	if err != nil {
		return nil, err
	}
	if !ok {
		return &Result{Lines: []string{"(nil)"}}, nil
	}
	return &Result{Lines: []string{val}}, nil
}

func (e *Executor) handleHGetAll(parts []string) (*Result, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'hgetall' command")
	}
	hash, ok, err := e.store.HGetAll(parts[1])
	if err != nil {
		return nil, err
	}
	if !ok {
		return &Result{Lines: []string{"(empty array)"}}, nil
	}
	fields := hash.HGetAll()
	lines := make([]string, 0, len(fields))
	for i := 0; i < len(fields); i += 2 {
		lines = append(lines, fmt.Sprintf("%d) \"%s\"", i+1, fields[i]))
		lines = append(lines, fmt.Sprintf("%d) \"%s\"", i+2, fields[i+1]))
	}
	return &Result{Lines: lines}, nil
}

func (e *Executor) handleHDel(parts []string) (*Result, error) {
	if len(parts) < 3 {
		return nil, fmt.Errorf("wrong number of arguments for 'hdel' command")
	}
	n, err := e.store.HDel(parts[1], parts[2:]...)
	if err != nil {
		return nil, err
	}
	return &Result{Lines: []string{fmt.Sprintf("(integer) %d", n)}}, nil
}
