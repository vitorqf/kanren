package web

import (
	"bufio"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLiveReloadOnFileChange: an external write to the cards directory pushes a
// "reload" SSE event to a connected client (WEB-03).
func TestLiveReloadOnFileChange(t *testing.T) {
	ts, dir, _ := newBoard(t)

	resp, err := http.Get(ts.URL + "/events")
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}

	got := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if line := sc.Text(); strings.HasPrefix(line, "data:") {
				got <- line
				return
			}
		}
	}()

	// Let the watcher register, then simulate a CLI edit / git pull.
	time.Sleep(200 * time.Millisecond)
	newFile := filepath.Join(dir, "cards", "0099-external.md")
	content := "---\nid: 99\ntitle: external\nstatus: todo\n---\nadded out of band\n"
	if err := os.WriteFile(newFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write external card: %v", err)
	}

	select {
	case line := <-got:
		if !strings.Contains(line, "reload") {
			t.Errorf("SSE data = %q, want reload", line)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no reload event after external file change")
	}
}

// TestEventsStreamOpens: /events opens a stream and sends the retry preamble
// (WEB-03 transport).
func TestEventsStreamOpens(t *testing.T) {
	ts, _, _ := newBoard(t)
	resp, err := http.Get(ts.URL + "/events")
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	got := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if line := sc.Text(); strings.HasPrefix(line, "retry:") {
				got <- line
				return
			}
		}
	}()
	select {
	case <-got: // retry preamble received
	case <-time.After(3 * time.Second):
		t.Fatal("no retry preamble on the SSE stream")
	}
}
