package rclonelib

import (
	"bufio"
	"fmt"
	"strings"
	"testing"
)

// feed builds a bufio.Reader over s for parseRcloneOutput.
func feed(s string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(s))
}

func TestParseRcloneOutput_ReturnsErrorTailNotProgress(t *testing.T) {
	mgr := NewManager()
	mgr.Add("t1", "src", "dst")

	input := strings.Join([]string{
		"Transferred:   1.234 GiB / 5.678 GiB, 22%, 10 MiB/s, ETA 1m30s",
		`ERROR : movie.mkv: Failed to copy: Put "http://ipad": context deadline exceeded`,
		"Transferred:   2.000 GiB / 5.678 GiB, 35%, 10 MiB/s, ETA 1m",
		"Failed to copy: 1 error(s)",
	}, "\n")

	tail := parseRcloneOutput(feed(input), "t1", mgr)

	// Progress must have been applied.
	if tr, ok := mgr.Get("t1"); !ok || tr.Progress != 35 {
		t.Fatalf("expected progress 35 from last stats line, got %+v (ok=%v)", tr, ok)
	}

	// Tail must contain the diagnostic lines and none of the progress lines.
	joined := strings.Join(tail, "\n")
	if !strings.Contains(joined, "context deadline exceeded") {
		t.Errorf("expected error line in tail, got %q", joined)
	}
	if !strings.Contains(joined, "1 error(s)") {
		t.Errorf("expected summary line in tail, got %q", joined)
	}
	for _, l := range tail {
		if strings.HasPrefix(l, "Transferred:") {
			t.Errorf("progress line leaked into diagnostic tail: %q", l)
		}
	}
}

func TestParseRcloneOutput_TailIsBounded(t *testing.T) {
	mgr := NewManager()
	mgr.Add("t1", "src", "dst")

	var b strings.Builder
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&b, "ERROR line %d\n", i)
	}

	tail := parseRcloneOutput(feed(b.String()), "t1", mgr)

	if len(tail) != 10 {
		t.Fatalf("expected tail bounded to 10, got %d", len(tail))
	}
	// Ring buffer keeps the most recent lines.
	if tail[0] != "ERROR line 90" || tail[9] != "ERROR line 99" {
		t.Errorf("expected last 10 lines (90..99), got first=%q last=%q", tail[0], tail[9])
	}
}
