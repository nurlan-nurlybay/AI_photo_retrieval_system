package seaweedfs

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalFS struct {
	Root       string // e.g. "./var/uploads"
	PublicBase string // e.g. "http://localhost:8080/uploads"
}

// NewLocalFS ensures the root exists
func NewLocalFS(root, publicBase string) (*LocalFS, error) {
	if root == "" {
		root = "./var/uploads"
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &LocalFS{Root: root, PublicBase: strings.TrimRight(publicBase, "/")}, nil
}

// Put writes the reader to <Root>/<key> and returns PublicBase + "/" + key
// key must be a POSIX-ish path (no traversal)
// Hash the content to make idempotent writes fast
func (fs *LocalFS) Put(_ context.Context, key string, r *bytes.Reader) (string, error) {
	safeKey, err := sanitizeKey(key)
	if err != nil {
		return "", err
	}
	full := filepath.Join(fs.Root, safeKey)

	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}

	// Optional: short-circuit if file already exists with same content
	if existsSame(full, r) {
		return fs.publicURL(safeKey), nil
	}
	// reset after existsSame read
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	tmp := full + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, full); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}

	return fs.publicURL(safeKey), nil
}

func (fs *LocalFS) Delete(_ context.Context, key string) error {
	safeKey, err := sanitizeKey(key)
	if err != nil {
		return err
	}
	full := filepath.Join(fs.Root, safeKey)
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return err
	}
	// Best-effort: clean empty parent dirs
	_ = tryRemoveEmptyParents(filepath.Dir(full), fs.Root)
	return nil
}

func (fs *LocalFS) publicURL(key string) string {
	// force forward slashes for URLs
	key = filepath.ToSlash(key)
	return fs.PublicBase + "/" + key
}

func sanitizeKey(key string) (string, error) {
	key = strings.TrimLeft(key, "/")
	if key == "" || strings.Contains(key, "..") || strings.ContainsAny(key, `\`) {
		return "", fmt.Errorf("invalid key")
	}
	return filepath.Clean(key), nil
}

func existsSame(path string, r *bytes.Reader) bool {
	// fast path: if file missing, no match
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	if st.Size() != int64(r.Len()) {
		return false
	}
	// compare short hash
	h1 := sha1.New()
	_, _ = io.Copy(h1, bytes.NewReader(mustReadAll(r)))
	sumUpload := hex.EncodeToString(h1.Sum(nil))

	fd, err := os.Open(path)
	if err != nil {
		return false
	}
	defer fd.Close()
	h2 := sha1.New()
	_, _ = io.Copy(h2, fd)
	sumFile := hex.EncodeToString(h2.Sum(nil))
	return sumUpload == sumFile
}

func mustReadAll(r *bytes.Reader) []byte {
	pos, _ := r.Seek(0, io.SeekCurrent)
	defer r.Seek(pos, io.SeekStart)
	all, _ := io.ReadAll(bytes.NewReader(peekAll(r)))
	return all
}

func peekAll(r *bytes.Reader) []byte {
	pos, _ := r.Seek(0, io.SeekCurrent)
	defer r.Seek(pos, io.SeekStart)
	n := r.Len()
	buf := make([]byte, n)
	_, _ = r.ReadAt(buf, pos)
	return buf
}

func tryRemoveEmptyParents(dir, stopAt string) error {
	for {
		if dir == stopAt || dir == "." || dir == string(filepath.Separator) {
			return nil
		}
		_ = os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}
