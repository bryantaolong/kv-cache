package cli

import (
	"fmt"
	"strconv"
)

func (c *CLI) handleLPush(parts []string) error {
	if len(parts) < 3 {
		return fmt.Errorf("wrong number of arguments for 'lpush' command")
	}
	n, err := c.store.LPush(parts[1], parts[2:]...)
	if err != nil {
		return err
	}
	if !c.loading {
		fmt.Fprintf(c.writer, "(integer) %d\n", n)
	}
	return nil
}

func (c *CLI) handleRPush(parts []string) error {
	if len(parts) < 3 {
		return fmt.Errorf("wrong number of arguments for 'rpush' command")
	}
	n, err := c.store.RPush(parts[1], parts[2:]...)
	if err != nil {
		return err
	}
	if !c.loading {
		fmt.Fprintf(c.writer, "(integer) %d\n", n)
	}
	return nil
}

func (c *CLI) handleLPop(parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("wrong number of arguments for 'lpop' command")
	}
	val, ok, err := c.store.LPop(parts[1])
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintln(c.writer, "(nil)")
		return nil
	}
	fmt.Fprintf(c.writer, "\"%s\"\n", val)
	return nil
}

func (c *CLI) handleRPop(parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("wrong number of arguments for 'rpop' command")
	}
	val, ok, err := c.store.RPop(parts[1])
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintln(c.writer, "(nil)")
		return nil
	}
	fmt.Fprintf(c.writer, "\"%s\"\n", val)
	return nil
}

func (c *CLI) handleLRange(parts []string) error {
	if len(parts) < 4 {
		return fmt.Errorf("wrong number of arguments for 'lrange' command")
	}
	start, err1 := strconv.Atoi(parts[2])
	stop, err2 := strconv.Atoi(parts[3])
	if err1 != nil || err2 != nil {
		return fmt.Errorf("invalid range")
	}
	vals, err := c.store.LRange(parts[1], start, stop)
	if err != nil {
		return err
	}
	if len(vals) == 0 {
		fmt.Fprintln(c.writer, "(empty array)")
		return nil
	}
	for i, v := range vals {
		fmt.Fprintf(c.writer, "%d) \"%s\"\n", i+1, v)
	}
	return nil
}

func (c *CLI) handleLLen(parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("wrong number of arguments for 'llen' command")
	}
	n, err := c.store.LLen(parts[1])
	if err != nil {
		return err
	}
	fmt.Fprintf(c.writer, "(integer) %d\n", n)
	return nil
}
