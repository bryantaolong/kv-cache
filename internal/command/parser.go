package command

import "strings"

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
