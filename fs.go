package ungocheck

import "os"

type File interface {
	Close() error
	Stat() (os.FileInfo, error)
	Readdir(int) ([]os.FileInfo, error)
	Read([]byte) (int, error)
	Write([]byte) (int, error)
}

type Filesystem interface {
	Open(string) (File, error)
	OpenFile(string, int, os.FileMode) (File, error)
	Remove(string) error
	Stat(string) (os.FileInfo, error)
}

type fs struct{}

func (fs) Open(name string) (File, error) {
	return os.Open(name)
}

func (fs) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	return os.OpenFile(name, flag, perm)
}

func (fs) Remove(name string) error {
	return os.Remove(name)
}

func (fs) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

var DefaultFilesystem Filesystem = fs{}
