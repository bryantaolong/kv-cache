package command

import (
	"reflect"
	"testing"
)

func TestSerializeArgsRoundTrip(t *testing.T) {
	cases := [][]string{
		{"SET", "name", "alice"},
		{"SET", "name", "alice jones"},
		{"SET", "msg", `hello "world"`},
		{"SET", "path", `C:\dir\file`},
		{"SET", "key", ""},
		{"SET", "key", "a\tb"},
		{"SET", "key", "a\nb"},
		{"HSET", "user:1", "name", "John Doe"},
		{"LPUSH", "mylist", "hello world", "foo", "bar"},
		{"SADD", "myset", "a member", "b"},
		{"ZADD", "z", "1.5", "a member"},
	}

	for _, args := range cases {
		line := SerializeArgs(args)
		got := ParseArgs(line)
		if !reflect.DeepEqual(got, args) {
			t.Errorf("round-trip mismatch:\n  input: %#v\n  line:  %q\n  got:   %#v", args, line, got)
		}
	}
}

func TestSerializeArgsLeavesSafeArgsUnquoted(t *testing.T) {
	if got := SerializeArgs([]string{"SET", "key", "value"}); got != "SET key value" {
		t.Errorf("expected unquoted output, got %q", got)
	}
}

func TestSerializeArgsQuotesSpacesAndEscapes(t *testing.T) {
	got := SerializeArgs([]string{"SET", "msg", `hello "world"`})
	want := `SET msg "hello \"world\""`
	if got != want {
		t.Errorf("SerializeArgs = %q, want %q", got, want)
	}
}
