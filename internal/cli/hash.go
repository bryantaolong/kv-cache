package cli

import "fmt"

func (c *CLI) handleHSet(parts []string) error {
	if len(parts) < 4 {
		return fmt.Errorf("wrong number of arguments for 'hset' command")
	}
	n, err := c.store.HSet(parts[1], parts[2], parts[3])
	if err != nil {
		return err
	}
	if !c.loading {
		fmt.Fprintf(c.writer, "(integer) %d\n", n)
	}
	return nil
}

func (c *CLI) handleHGet(parts []string) error {
	if len(parts) < 3 {
		return fmt.Errorf("wrong number of arguments for 'hget' command")
	}
	val, ok, err := c.store.HGet(parts[1], parts[2])
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

func (c *CLI) handleHGetAll(parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("wrong number of arguments for 'hgetall' command")
	}
	hash, ok, err := c.store.HGetAll(parts[1])
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintln(c.writer, "(empty array)")
		return nil
	}
	fields := hash.HGetAll()
	for i := 0; i < len(fields); i += 2 {
		fmt.Fprintf(c.writer, "%d) \"%s\"\n", i+1, fields[i])
		fmt.Fprintf(c.writer, "%d) \"%s\"\n", i+2, fields[i+1])
	}
	return nil
}

func (c *CLI) handleHDel(parts []string) error {
	if len(parts) < 3 {
		return fmt.Errorf("wrong number of arguments for 'hdel' command")
	}
	n, err := c.store.HDel(parts[1], parts[2:]...)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.writer, "(integer) %d\n", n)
	return nil
}
