package uploadstore

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStoreRejectsPathsOutsideRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewLocal(root)

	if err := store.Delete(context.Background(), outside); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside file was touched: %v", err)
	}

	err := store.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/upload", nil), Object{Path: outside})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected outside path to be rejected, got %v", err)
	}
}

func TestLocalStoreServesRelativePathsInsideRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store := NewLocal(root)
	saved, err := store.Save(context.Background(), strings.NewReader("hello"), SaveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(root, saved.Path)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	err = store.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/upload", nil), Object{Path: relative})
	if err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || recorder.Body.String() != "hello" {
		t.Fatalf("unexpected local serve response: %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestLocalStoreServesSavedPathFromRelativeRoot(t *testing.T) {
	t.Chdir(t.TempDir())
	store := NewLocal(filepath.Join("data", "uploads"))
	saved, err := store.Save(context.Background(), strings.NewReader("hello"), SaveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(saved.Path, filepath.Join("data", "uploads")+string(os.PathSeparator)) {
		t.Fatalf("expected saved path under relative root, got %q", saved.Path)
	}

	recorder := httptest.NewRecorder()
	err = store.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/upload", nil), Object{Path: saved.Path})
	if err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || recorder.Body.String() != "hello" {
		t.Fatalf("unexpected local serve response: %d %q", recorder.Code, recorder.Body.String())
	}
}
