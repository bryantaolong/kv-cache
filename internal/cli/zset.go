package cli

import (
	"fmt"
	"strconv"
)

func (c *CLI) handleZAdd(parts []string) error {
	if len(parts) < 4 {
		return fmt.Errorf("wrong number of arguments for 'zadd' command")
	}
	score, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return fmt.Errorf("invalid score")
	}
	n, err := c.store.ZAdd(parts[1], score, parts[3])
	if err != nil {
		return err
	}
	if !c.loading {
		fmt.Fprintf(c.writer, "(integer) %d\n", n)
	}
	return nil
}

func (c *CLI) handleZRange(parts []string) error {
	if len(parts) < 4 {
		return fmt.Errorf("wrong number of arguments for 'zrange' command")
	}
	start, err1 := strconv.Atoi(parts[2])
	stop, err2 := strconv.Atoi(parts[3])
	if err1 != nil || err2 != nil {
		return fmt.Errorf("invalid range")
	}
	members, err := c.store.ZRange(parts[1], start, stop)
	if err != nil {
		return err
	}
	if len(members) == 0 {
		fmt.Fprintln(c.writer, "(empty array)")
		return nil
	}
	for i, m := range members {
		fmt.Fprintf(c.writer, "%d) \"%s\"\n", i+1, m.Member)
		fmt.Fprintf(c.writer, "   score: %g\n", m.Score)
	}
	return nil
}

func (c *CLI) handleZCard(parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("wrong number of arguments for 'zcard' command")
	}
	n, err := c.store.ZCard(parts[1])
	if err != nil {
		return err
	}
	fmt.Fprintf(c.writer, "(integer) %d\n", n)
	return nil
}
