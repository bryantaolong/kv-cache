package cli

import "fmt"

func (c *CLI) handleSAdd(parts []string) error {
	if len(parts) < 3 {
		return fmt.Errorf("wrong number of arguments for 'sadd' command")
	}
	n, err := c.store.SAdd(parts[1], parts[2:]...)
	if err != nil {
		return err
	}
	if !c.loading {
		fmt.Fprintf(c.writer, "(integer) %d\n", n)
	}
	return nil
}

func (c *CLI) handleSMembers(parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("wrong number of arguments for 'smembers' command")
	}
	members, err := c.store.SMembers(parts[1])
	if err != nil {
		return err
	}
	if len(members) == 0 {
		fmt.Fprintln(c.writer, "(empty array)")
		return nil
	}
	for i, m := range members {
		fmt.Fprintf(c.writer, "%d) \"%s\"\n", i+1, m)
	}
	return nil
}

func (c *CLI) handleSCard(parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("wrong number of arguments for 'scard' command")
	}
	n, err := c.store.SCard(parts[1])
	if err != nil {
		return err
	}
	fmt.Fprintf(c.writer, "(integer) %d\n", n)
	return nil
}
