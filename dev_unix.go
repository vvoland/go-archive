//go:build !windows && !freebsd

package archive

import "golang.org/x/sys/unix"

func mknod(path string, mode uint32, dev uint64) error {
	return unix.Mknod(path, mode, int(dev))
}

func mknodat(dirFd int, path string, mode uint32, dev uint64) error {
	return unix.Mknodat(dirFd, path, mode, int(dev))
}
