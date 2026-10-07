//go:build darwin || freebsd || netbsd

package archive

import (
	"archive/tar"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/skip"
)

// TestLsetxattrInRootRejectsSpecialFiles guards against opening special files
// to set attributes; for device nodes, opening can trigger driver side effects.
// A FIFO stands in for devices because creating it needs no privileges.
func TestLsetxattrInRootRejectsSpecialFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	assert.NilError(t, unix.Mkfifo(filepath.Join(dir, "fifo"), 0o600))
	root, err := os.OpenRoot(dir)
	assert.NilError(t, err)
	defer root.Close()

	err = lsetxattrInRoot(root, "fifo", "user.go_archive_root", []byte("archive"), 0)
	assert.Assert(t, errors.Is(err, unix.ENOTSUP), "got %v", err)
}

// TestUntarDestinationRootXattrsOnWriteOnlyFile catches xattr failures on
// entries the owner cannot read, which pathname-based setxattr allows.
func TestUntarDestinationRootXattrsOnWriteOnlyFile(t *testing.T) {
	t.Parallel()
	skip.If(t, os.Getuid() == 0, "root bypasses read permission checks")
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	assert.NilError(t, err)
	defer root.Close()

	const attr = "user.go_archive_root"
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	assert.NilError(t, tw.WriteHeader(&tar.Header{
		Name:       "file",
		Typeflag:   tar.TypeReg,
		Mode:       0o200,
		PAXRecords: map[string]string{paxSchilyXattr + attr: "archive"},
	}))
	assert.NilError(t, tw.Close())
	err = Untar(&buf, "", &TarOptions{NoLchown: true, DestinationRoot: root})
	skip.If(t, errors.Is(err, unix.ENOTSUP), "filesystem does not support xattrs")
	assert.NilError(t, err)

	path := filepath.Join(dir, "file")
	fi, err := os.Lstat(path)
	assert.NilError(t, err)
	assert.Equal(t, fi.Mode().Perm(), os.FileMode(0o200))
	// Reading attributes needs read access.
	assert.NilError(t, os.Chmod(path, 0o600))
	actual, err := lgetxattr(path, attr)
	assert.NilError(t, err)
	assert.Equal(t, string(actual), "archive")
}
