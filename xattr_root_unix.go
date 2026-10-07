//go:build darwin || freebsd || netbsd

package archive

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func lsetxattrInRoot(root *os.Root, name, attr string, data []byte, flags int) error {
	// Open only regular files and directories: opening a device node can
	// trigger driver side effects.
	fi, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() && !fi.IsDir() {
		return wrapPathError("fsetxattr", name, attr, unix.ENOTSUP)
	}
	// Do not fall back to root.Name(): a rename may have replaced that directory.
	file, err := openForXattr(root, name, fi.Mode())
	if err != nil {
		if errors.Is(err, unix.ELOOP) {
			return wrapPathError("fsetxattr", name, attr, unix.ENOTSUP)
		}
		return err
	}
	defer file.Close()
	return wrapPathError("fsetxattr", name, attr, unix.Fsetxattr(int(file.Fd()), attr, data, flags)) // #nosec G115 -- Required descriptor conversion.
}

// openForXattr opens name for fsetxattr, which needs a descriptor, whereas
// pathname-based setxattr needs only write access.
// Owner read access is granted only for the open, for example for a write-only
// file, and the mode is restored before attributes are set so their permission
// checks still apply.
func openForXattr(root *os.Root, name string, mode os.FileMode) (*os.File, error) {
	open := func() (*os.File, error) {
		return root.OpenFile(name, os.O_RDONLY|unix.O_NONBLOCK|unix.O_NOFOLLOW, 0)
	}
	file, err := open()
	if !errors.Is(err, unix.EACCES) || mode&0o400 != 0 {
		return file, err
	}
	if chmodNoSymlink(root, name, mode|0o400, nil) != nil {
		return nil, err
	}
	file, err = open()
	if restoreErr := chmodNoSymlink(root, name, mode, nil); restoreErr != nil {
		if file != nil {
			_ = file.Close()
		}
		return nil, restoreErr
	}
	return file, err
}
