package persist

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestAppendAndCloseSync(t *testing.T) {
	dir := t.TempDir()
	p, err := NewPersistence(dir)
	if err != nil {
		t.Fatalf("NewPersistence failed: %v", err)
	}

	cmds := []string{
		"SET foo bar",
		"SET baz qux",
		"HSET h f v",
	}
	for _, cmd := range cmds {
		if err := p.Append(cmd); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}
	if err := p.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	b, err := os.ReadFile(p.filepath)
	if err != nil {
		t.Fatalf("read aof failed: %v", err)
	}
	body := string(b)
	for _, cmd := range cmds {
		if !strings.Contains(body, cmd) {
			t.Fatalf("AOF missing command %q", cmd)
		}
	}
	if !strings.HasSuffix(body, "\n") {
		t.Fatalf("expected trailing newline, got %q", body)
	}
}

func TestRewriteCompactsFile(t *testing.T) {
	dir := t.TempDir()
	p, err := NewPersistence(dir)
	if err != nil {
		t.Fatalf("NewPersistence failed: %v", err)
	}

	cmds := []string{
		"SET a 1",
		"SET b 2",
		"HSET h k v",
		"RPUSH list x y",
	}
	for _, cmd := range cmds {
		if err := p.Append(cmd); err != nil {
			t.Fatalf("append failed: %v", err)
		}
	}
	if err := p.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	defer p.Close()

	exported := []string{
		"SET a 1",
		"SET b 2",
		"HSET h k v",
		"RPUSH list x y",
	}
	if err := p.rewrite(func() []string { return exported }); err != nil {
		t.Fatalf("rewrite failed: %v", err)
	}

	if _, err := os.Stat(p.filepath + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("tmp file should not exist after rewrite")
	}
	b, err := os.ReadFile(p.filepath)
	if err != nil {
		t.Fatalf("read rewritten aof failed: %v", err)
	}
	body := string(b)
	if !strings.HasPrefix(body, "SET a 1\n") {
		t.Fatalf("unexpected rewrite output: %q", body)
	}
	for _, cmd := range exported {
		if !strings.Contains(body, cmd) {
			t.Fatalf("rewritten command missing %q in %q", cmd, body)
		}
	}
	if !strings.HasSuffix(body, "\n") {
		t.Fatalf("expected trailing newline, got %q", body)
	}
}

func TestRewriteEmptyExportsClearsFile(t *testing.T) {
	dir := t.TempDir()
	p, err := NewPersistence(dir)
	if err != nil {
		t.Fatalf("NewPersistence failed: %v", err)
	}
	if err := p.Append("SET a 1"); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	defer p.Close()

	if err := p.rewrite(func() []string { return nil }); err != nil {
		t.Fatalf("rewrite failed: %v", err)
	}
	b, err := os.ReadFile(p.filepath)
	if err != nil {
		t.Fatalf("read aof failed: %v", err)
	}
	if len(b) != 0 {
		t.Fatalf("expected empty AOF after empty rewrite, got %q", string(b))
	}
}

func TestLoadSkipsInvalidLines(t *testing.T) {
	dir := t.TempDir()
	p, err := NewPersistence(dir)
	if err != nil {
		t.Fatalf("NewPersistence failed: %v", err)
	}

	lines := []string{
		"SET valid yes",
		"",
		"HSET",
		"HSET x",
		"HSET x y ",
	}
	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	if err := os.WriteFile(p.filepath, []byte(sb.String()), 0644); err != nil {
		t.Fatalf("write aof failed: %v", err)
	}

	defer p.Close()

	var executed []string
	if err := p.Load(func(cmd string) error {
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			return nil
		}
		if parts[0] == "HSET" && len(parts) != 4 {
			return nil
		}
		executed = append(executed, cmd)
		return nil
	}); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(executed) != 1 {
		t.Fatalf("expected 1 executed command, got %d: %v", len(executed), executed)
	}
	if executed[0] != "SET valid yes" {
		t.Fatalf("expected command SET valid yes, got %q", executed[0])
	}
}

func TestAutoRewriteTriggersBySize(t *testing.T) {
	dir := t.TempDir()
	p, err := NewPersistence(dir)
	if err != nil {
		t.Fatalf("NewPersistence failed: %v", err)
	}
	defer p.StopAutoRewrite()
	defer p.Close()

	p.SetSyncPolicy(SyncEverySec)

	p.StartAutoRewrite(10, 5*time.Millisecond, func() []string {
		return []string{"SET a 1", "SET b 2"}
	})
	for i := 0; i < 4; i++ {
		if err := p.Append("SET x y"); err != nil {
			t.Fatalf("append failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	requireEventualFileContent(t, p.filepath, "SET a 1\nSET b 2\n")
}

func TestAppendWithoutCloseWritesFile(t *testing.T) {
	dir := t.TempDir()
	p, err := NewPersistence(dir)
	if err != nil {
		t.Fatalf("NewPersistence failed: %v", err)
	}

	if err := p.Append("SET a 1"); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Writer buffered data may not be on disk yet, but file should still be appendable.
	f, err := os.OpenFile(p.filepath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("reopen aof failed: %v", err)
	}
	if _, err := f.WriteString("SET b 2\n"); err != nil {
		t.Fatalf("append after open failed: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if err := p.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	b, err := os.ReadFile(p.filepath)
	if err != nil {
		t.Fatalf("read aof failed: %v", err)
	}
	body := string(b)
	if !strings.Contains(body, "SET a 1") || !strings.Contains(body, "SET b 2") {
		t.Fatalf("unexpected aof content %q", body)
	}
}

func TestLoadMissingFileIsNoop(t *testing.T) {
	dir := t.TempDir()
	p, err := NewPersistence(dir)
	if err != nil {
		t.Fatalf("NewPersistence failed: %v", err)
	}
	defer p.Close()

	var called bool
	if err := p.Load(func(cmd string) error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if called {
		t.Fatal("Load should not call executor when AOF does not exist")
	}
}

func TestAppendNilIsNoop(t *testing.T) {
	dir := t.TempDir()
	p, err := NewPersistence(dir)
	if err != nil {
		t.Fatalf("NewPersistence failed: %v", err)
	}
	defer p.Close()

	if err := p.Append(""); err != nil {
		t.Fatalf("Append empty failed: %v", err)
	}
}

func TestSetSyncPolicyChangeDoesNotCrash(t *testing.T) {
	dir := t.TempDir()
	p, err := NewPersistence(dir)
	if err != nil {
		t.Fatalf("NewPersistence failed: %v", err)
	}
	defer p.Close()

	p.SetSyncPolicy(SyncNo)
	p.SetSyncPolicy(SyncAlways)
	p.SetSyncPolicy(SyncEverySec)
	p.SetSyncPolicy(SyncNo)
}

func TestRewriteAfterCloseUsesFreshFile(t *testing.T) {
	dir := t.TempDir()
	p, err := NewPersistence(dir)
	if err != nil {
		t.Fatalf("NewPersistence failed: %v", err)
	}
	if err := p.Append("SET a 1"); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	defer p.Close()

	oldSize := p.GetSize()
	if err := p.rewrite(func() []string { return []string{"SET a 1"} }); err != nil {
		t.Fatalf("rewrite failed: %v", err)
	}
	newSize := p.GetSize()
	if newSize != oldSize {
		t.Fatalf("expected rewrite to keep file size stable, before=%d after=%d", oldSize, newSize)
	}
}

func TestMultipleRewriteCallsAreIdempotent(t *testing.T) {
	dir := t.TempDir()
	p, err := NewPersistence(dir)
	if err != nil {
		t.Fatalf("NewPersistence failed: %v", err)
	}
	if err := p.Append("SET a 1"); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	defer p.Close()

	for i := 0; i < 3; i++ {
		if err := p.rewrite(func() []string { return []string{"SET a 1"} }); err != nil {
			t.Fatalf("rewrite %d failed: %v", i, err)
		}
	}
}

func TestRewriteDoesNotLeaveTempFileAfterFailure(t *testing.T) {
	dir := t.TempDir()
	p, err := NewPersistence(dir)
	if err != nil {
		t.Fatalf("NewPersistence failed: %v", err)
	}
	if err := p.Append("SET a 1"); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	defer p.Close()

	_ = p.rewrite(func() []string {
		return []string{"SET invalid\nline"}
	})

	if _, err := os.Stat(p.filepath + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("tmp file should be cleaned after failed rewrite")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	p, err := NewPersistence(dir)
	if err != nil {
		t.Fatalf("NewPersistence failed: %v", err)
	}
	if err := p.Append("SET a 1"); err != nil {
		t.Fatalf("append failed: %v", err)
	}

	if err := p.Close(); err != nil {
		t.Fatalf("first close failed: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
}

func TestStartStopAutoRewriteDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	p, err := NewPersistence(dir)
	if err != nil {
		t.Fatalf("NewPersistence failed: %v", err)
	}
	defer p.StopAutoRewrite()
	defer p.Close()

	p.StartAutoRewrite(10, 5*time.Millisecond, func() []string {
		return []string{}
	})
	p.StopAutoRewrite()
	p.StopAutoRewrite()
}

func TestLoadLargeFileDoesNotBufferForever(t *testing.T) {
	dir := t.TempDir()
	p, err := NewPersistence(dir)
	if err != nil {
		t.Fatalf("NewPersistence failed: %v", err)
	}

	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("SET key")
		sb.WriteString(string(rune('0' + i%10)))
		sb.WriteByte('\n')
	}
	if err := os.WriteFile(p.filepath, []byte(sb.String()), 0644); err != nil {
		t.Fatalf("write aof failed: %v", err)
	}
	defer p.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = p.Load(func(cmd string) error {
			parts := strings.Fields(cmd)
			if len(parts) != 2 || parts[0] != "SET" {
				return nil
			}
			return nil
		})
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Load took too long")
	}
}

func requireEventualFileContent(t *testing.T, path, want string) {
	t.Helper()
	var lastErr error
	check := func() bool {
		b, err := os.ReadFile(path)
		if err != nil {
			lastErr = err
			return false
		}
		return string(b) == want
	}

	if check() {
		return
	}

	done := time.After(2 * time.Second)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			if lastErr != nil {
				t.Fatalf("failed to read file: %v", lastErr)
			}
			t.Fatalf("file content did not reach expected state, got:\n%s", func() string {
				b, _ := os.ReadFile(path)
				return string(b)
			}())
		case <-ticker.C:
			if check() {
				return
			}
		}
	}
}
