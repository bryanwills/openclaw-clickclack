package uploadstore

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Local struct {
	dir string
}

func NewLocal(dir string) *Local {
	return &Local{dir: dir}
}

func (s *Local) Save(_ context.Context, body io.Reader, _ SaveOptions) (SavedObject, error) {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return SavedObject{}, err
	}
	tmp, err := os.CreateTemp(s.dir, "upload-*")
	if err != nil {
		return SavedObject{}, err
	}
	committed := false
	defer func() {
		_ = tmp.Close()
		if !committed {
			_ = os.Remove(tmp.Name())
		}
	}()
	size, err := io.Copy(tmp, body)
	if err != nil {
		return SavedObject{}, err
	}
	if err := tmp.Close(); err != nil {
		return SavedObject{}, err
	}
	committed = true
	return SavedObject{Path: tmp.Name(), ByteSize: size}, nil
}

func (s *Local) Delete(_ context.Context, path string) error {
	if path == "" {
		return nil
	}
	resolved, err := s.resolvePath(path)
	if err != nil {
		if err == ErrNotFound {
			return nil
		}
		return err
	}
	if err := os.Remove(resolved); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Local) ServeHTTP(w http.ResponseWriter, r *http.Request, object Object) error {
	resolved, err := s.resolvePath(object.Path)
	if err != nil {
		return err
	}
	http.ServeFile(w, r, resolved)
	return nil
}

func (s *Local) resolvePath(path string) (string, error) {
	if path == "" {
		return "", ErrNotFound
	}
	root, err := filepath.Abs(s.dir)
	if err != nil {
		return "", err
	}
	candidate := filepath.Clean(path)
	if !filepath.IsAbs(candidate) {
		if cwdCandidate, err := filepath.Abs(candidate); err == nil && isLocalPathInRoot(root, cwdCandidate) {
			return cwdCandidate, nil
		}
		candidate = filepath.Join(root, candidate)
	}
	if !isLocalPathInRoot(root, candidate) {
		return "", ErrNotFound
	}
	return candidate, nil
}

func isLocalPathInRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
