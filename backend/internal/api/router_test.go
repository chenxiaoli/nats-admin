package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestRouter_ServesStaticAndAPIPaths(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":  &fstest.MapFile{Data: []byte("<html>app</html>")},
		"assets/x.js": &fstest.MapFile{Data: []byte("x")},
	}

	r := NewRouter(Deps{
		JWTSecret:  []byte("test-secret-test-secret-test-secret-1234"),
		FrontendFS: fsys,
	})
	srv := httptest.NewServer(r)
	defer srv.Close()

	t.Run("static root", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "<html>app</html>" {
			t.Fatalf("got %q", body)
		}
	})

	t.Run("spa route falls back to index", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/tenants/abc")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "<html>app</html>" {
			t.Fatalf("expected spa fallback, got %q", body)
		}
	})

	t.Run("api path not intercepted by static", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/v1/tenants")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		// Without auth, middleware should 401 — proving the static handler
		// did not intercept the api path.
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 from auth middleware, got %d", resp.StatusCode)
		}
	})
}
