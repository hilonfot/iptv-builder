package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#EXTM3U\n#EXTINF:-1 ,CCTV1\nhttp://stream/cctv1.ts\n"))
	}))
	defer srv.Close()

	f := New(10 * time.Second)
	results := f.Fetch(context.Background(), []string{srv.URL})

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].URL != srv.URL {
		t.Errorf("URL = %s, want %s", results[0].URL, srv.URL)
	}
	if len(results[0].Content) == 0 {
		t.Error("content is empty")
	}
}

func TestFetch_PartialFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#EXTM3U\n"))
	}))
	defer srv.Close()

	f := New(10 * time.Second)
	results := f.Fetch(context.Background(), []string{
		srv.URL,
		"http://127.0.0.1:19999/nonexistent.m3u", // will fail
	})

	// Only the successful one should be returned.
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

func TestFetch_AllFail(t *testing.T) {
	f := New(5 * time.Second)
	results := f.Fetch(context.Background(), []string{
		"http://127.0.0.1:19999/fail1.m3u",
		"http://127.0.0.1:19999/fail2.m3u",
	})

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestFetch_EmptyURLList(t *testing.T) {
	f := New(10 * time.Second)
	results := f.Fetch(context.Background(), []string{})
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestFetch_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte("data"))
	}))
	defer srv.Close()

	f := New(100 * time.Millisecond) // very short timeout
	results := f.Fetch(context.Background(), []string{srv.URL})

	if len(results) != 0 {
		t.Errorf("expected 0 results due to timeout, got %d", len(results))
	}
}

func TestFetch_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	f := New(10 * time.Second)
	results := f.Fetch(context.Background(), []string{srv.URL})

	if len(results) != 0 {
		t.Errorf("expected 0 results for 404, got %d", len(results))
	}
}

func TestFetch_Concurrent(t *testing.T) {
	// Verify concurrent fetching works with multiple sources.
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("source1"))
	}))
	defer srv1.Close()

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("source2"))
	}))
	defer srv2.Close()

	f := New(10 * time.Second)
	results := f.Fetch(context.Background(), []string{srv1.URL, srv2.URL})

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
}
