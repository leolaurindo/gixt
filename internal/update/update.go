// Package update installs checksum-verified release binaries without modifying
// the current executable until download and extraction have succeeded.
package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Apply downloads asset and checksums.txt from a release, then replaces the
// resolved executable. The HTTP client and release URL are supplied by the caller.
func Apply(ctx context.Context, client *http.Client, releaseURL, asset, executable string) error {
	target, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return err
	}
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("executable is not a regular file: %s", target)
	}

	lockPath := filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".update-lock")
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("cannot lock update at %s (another update may be running): %w", lockPath, err)
	}
	_ = lock.Close()
	defer os.Remove(lockPath)

	dir, err := os.MkdirTemp(filepath.Dir(target), ".gixt-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	checksums, err := download(ctx, client, releaseURL+"/checksums.txt", 1<<20)
	if err != nil {
		return err
	}
	expected := ""
	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == asset {
			expected = fields[0]
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("checksum not found for %s", asset)
	}
	archive, err := download(ctx, client, releaseURL+"/"+asset, 64<<20)
	if err != nil {
		return err
	}
	if !strings.EqualFold(expected, fmt.Sprintf("%x", sha256.Sum256(archive))) {
		return fmt.Errorf("checksum verification failed for %s", asset)
	}

	name := "gixt"
	if strings.HasSuffix(asset, ".zip") {
		name += ".exe"
	}
	binary, err := extractBinary(archive, name)
	if err != nil {
		return err
	}
	staged := filepath.Join(dir, name)
	if err := os.WriteFile(staged, binary, 0o600); err != nil {
		return err
	}
	if err := os.Chmod(staged, info.Mode().Perm()); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return replaceExecutable(staged, target)
}

func download(ctx context.Context, client *http.Client, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: HTTP %d", url, resp.StatusCode)
	}
	return readLimited(resp.Body, limit)
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err == nil && int64(len(data)) > limit {
		err = fmt.Errorf("update payload exceeds %d bytes", limit)
	}
	return data, err
}

func extractBinary(data []byte, name string) ([]byte, error) {
	if name == "gixt.exe" {
		archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		for _, file := range archive.File {
			if file.Name != name {
				continue
			}
			if !file.Mode().IsRegular() {
				return nil, fmt.Errorf("archive entry %s is not a regular file", name)
			}
			r, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer r.Close()
			return readLimited(r, 128<<20)
		}
	} else {
		r, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer r.Close()
		archive := tar.NewReader(io.LimitReader(r, 128<<20))
		for {
			file, err := archive.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			if file.Name != name {
				continue
			}
			if file.Typeflag != tar.TypeReg && file.Typeflag != tar.TypeRegA {
				return nil, fmt.Errorf("archive entry %s is not a regular file", name)
			}
			return readLimited(archive, 128<<20)
		}
	}
	return nil, fmt.Errorf("archive does not contain %s", name)
}

// Unix uses one atomic rename. Windows first renames the running executable
// to a backup, which may remain locked until exit; the next update removes it.
func replaceExecutable(staged, target string) error {
	if runtime.GOOS != "windows" {
		return os.Rename(staged, target)
	}
	backup := filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".old")
	if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot remove previous update backup %s: %w", backup, err)
	}
	if err := os.Rename(target, backup); err != nil {
		return err
	}
	if err := os.Rename(staged, target); err != nil {
		if rollbackErr := os.Rename(backup, target); rollbackErr != nil {
			return fmt.Errorf("install failed: %v; rollback failed: %v; restore %s manually", err, rollbackErr, backup)
		}
		return fmt.Errorf("install failed; previous executable restored: %w", err)
	}
	_ = os.Remove(backup)
	return nil
}
