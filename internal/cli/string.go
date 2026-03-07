package cli

import (
	"fmt"
	"strconv"
	"time"
)

func (c *CLI) handleSet(parts []string) error {
	if len(parts) < 3 {
		return fmt.Errorf("wrong number of arguments for 'set' command")
	}
	key, value := parts[1], parts[2]
	ttl := time.Duration(0)
	if len(parts) >= 4 {
		sec, err := strconv.Atoi(parts[3])
		if err != nil {
			return fmt.Errorf("invalid TTL")
		}
		ttl = time.Duration(sec) * time.Second
	}
	if err := c.store.SetString(key, value, ttl); err != nil {
		return err
	}
	if !c.loading {
		fmt.Fprintln(c.writer, "OK")
	}
	return nil
}

func (c *CLI) handleGet(parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("wrong number of arguments for 'get' command")
	}
	val, ok := c.store.GetString(parts[1])
	if !ok {
		fmt.Fprintln(c.writer, "(nil)")
		return nil
	}
	fmt.Fprintf(c.writer, "\"%s\"\n", val)
	return nil
}
