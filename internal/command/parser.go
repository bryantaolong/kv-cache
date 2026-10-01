package command

import "strings"

// quoteNeeded 判断参数是否需要加引号才能无损往返：
// 空串、空格/制表符、换行符、双引号或反斜杠都需要引号。
func quoteNeeded(s string) bool {
	if s == "" {
		return true
	}
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t', '\n', '\r', '"', '\\':
			return true
		}
	}
	return false
}

// quoteArg 用双引号包裹参数，并对反斜杠、双引号和控制字符做转义。
func quoteArg(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(s[i])
		}
	}
	b.WriteByte('"')
	return b.String()
}

// SerializeArgs 将命令参数编码为单行文本，对需要引号的参数做转义。
// 它是 ParseArgs 的逆操作，用于把命令无损地写入 AOF 文件。
func SerializeArgs(args []string) string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if quoteNeeded(a) {
			a = quoteArg(a)
		}
		out = append(out, a)
	}
	return strings.Join(out, " ")
}

// ParseArgs 解析命令行参数，支持双引号包裹的字符串和反斜杠转义。
// 它是 SerializeArgs 的逆操作。
func ParseArgs(line string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false
	started := false

	flush := func() {
		if started {
			args = append(args, current.String())
			current.Reset()
			started = false
		}
	}

	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c == '"':
			started = true
			inQuotes = !inQuotes
		case c == '\\' && inQuotes && i+1 < len(line):
			started = true
			i++
			switch line[i] {
			case 'n':
				current.WriteByte('\n')
			case 'r':
				current.WriteByte('\r')
			case 't':
				current.WriteByte('\t')
			default:
				current.WriteByte(line[i])
			}
		case (c == ' ' || c == '\t') && !inQuotes:
			flush()
		default:
			started = true
			current.WriteByte(c)
		}
	}
	flush()

	return args
}
