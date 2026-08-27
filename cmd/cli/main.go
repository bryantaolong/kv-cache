package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"kv-cache/internal/transport"
)

const defaultAddr = ":27926"

// ParseArgs 解析命令行参数，支持引号包裹的字符串
func ParseArgs(line string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if ch == '"' {
			if inQuotes {
				args = append(args, current.String())
				current.Reset()
				inQuotes = false
			} else {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
				inQuotes = true
			}
		} else if ch == ' ' && !inQuotes {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

func main() {
	addr := flag.String("addr", defaultAddr, "server 地址")
	flag.Parse()

	client, err := transport.NewClient(*addr)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout

	fmt.Fprintf(writer, "Connected to kv-cache at %s\n", *addr)

	for {
		fmt.Fprint(writer, "kv-cache> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(writer, "(error) %v\n", err)
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := ParseArgs(line)
		if len(parts) == 0 {
			continue
		}

		cmd := strings.ToUpper(parts[0])
		if cmd == "QUIT" || cmd == "EXIT" {
			fmt.Fprintln(writer, "Bye!")
			return
		}

		results, err := client.Execute(parts)
		if err != nil {
			fmt.Fprintf(writer, "(error) ERR %v\n", err)
			continue
		}

		for _, r := range results {
			fmt.Fprintln(writer, r)
		}
	}
}
