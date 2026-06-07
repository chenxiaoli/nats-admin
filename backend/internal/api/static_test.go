package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestStaticHandler_ServesEmbeddedFile(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":     &fstest.MapFile{Data: []byte("<html>root</html>")},
		"assets/app.js":  &fstest.MapFile{Data: []byte("console.log(1)")},
		"assets/app.css": &fstest.MapFile{Data: []byte("body{}")},
	}

	handler := staticHandler(fsys)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/index.html")
	if err != nil {
		t.Fatalf("get index: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "<html>root</html>" {
		t.Fatalf("unexpected body: %q", body)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestStaticHandler_SPAFallback(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>spa</html>")},
	}

	handler := staticHandler(fsys)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	for _, path := range []string{"/", "/tenants", "/tenants/123?tab=creds"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("get %s: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != "<html>spa</html>" {
			t.Fatalf("path %s: expected spa fallback, got %q", path, body)
		}
	}
}

