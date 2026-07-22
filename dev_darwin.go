//go:build darwin

package archive

import (
	"os"

	"golang.org/x/sys/unix"
)

func mknod(path string, mode uint32, dev uint64) error {
	return unix.Mknod(path, mode, int(dev))
}

func mknodInRoot(root *os.Root, path string, mode uint32, dev uint64) error {
	abs, err := fsRootPath(root.Name(), path)
	if err != nil {
		return err
	}
	return unix.Mknod(abs, mode, int(dev))
}
