package ungocheck

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type MultiError []error

func (me MultiError) Err() error {
	if len(me) == 0 {
		return nil
	}
	var s []string
	for _, err := range me {
		s = append(s, err.Error())
	}
	return errors.New(strings.Join(s, "\n"))
}

func (me MultiError) Error() (s string) {
	if err := me.Err(); err != nil {
		s = err.Error()
	}
	return
}

func (me *MultiError) Consume(v ...interface{}) *MultiError {
	for _, v := range v {
		if err, ok := v.(error); ok && err != nil {
			*me = append(*me, err)
		} else if err, ok := v.(MultiError); ok && err != nil && len(err) != 0 {
			for _, err := range err {
				*me = append(*me, err)
			}
		}
	}
	return me
}

type Ungocheck struct {
	FS Filesystem
}

func New() *Ungocheck {
	return &Ungocheck{}
}

func (u Ungocheck) Packages(args []string) (pkgs []string) {
	// TODO(rjeczalik): filter out space-separated values for -test.* flags
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			pkgs = append(pkgs, arg)
		}
	}
	if len(pkgs) == 0 {
		pkgs = append(pkgs, ".")
	}
	return
}

func istestfile(name string) bool {
	return strings.HasSuffix(name, "_test.go") && !strings.HasSuffix(name, "_ungocheck_test.go")
}

var vcs = map[string]struct{}{
	".bzr": {},
	".git": {},
	".hg":  {},
	".svn": {},
}

func isnotvcs(name string) bool {
	_, ok := vcs[name]
	return !ok
}

func lookup(fs Filesystem, paths []string, pkg string) (dir string, err error) {
	for _, path := range paths {
		dir = filepath.Join(path, "src", pkg)
		if _, err = fs.Stat(dir); err == nil {
			break
		}
	}
	return
}

func (u Ungocheck) Files(pkgs []string) (files []string, err error) {
	var (
		wd, dir string
		f       File
		fi      []os.FileInfo
		fs      = u.fs()
		dirs    = make(map[string]struct{})
		paths   = strings.Split(os.Getenv("GOPATH"), string(os.PathListSeparator))
	)
	if wd, err = os.Getwd(); err != nil {
		return
	}
	for _, pkg := range pkgs {
		switch {
		case pkg == ".":
			dirs[wd] = struct{}{}
		case strings.HasSuffix(pkg, "/..."):
			var glob []string
			pkg = pkg[:len(pkg)-4]
			if pkg == "." {
				glob = append(glob, wd)
			} else {
				if dir, err = lookup(fs, paths, pkg); err != nil {
					return
				}
				glob = append(glob, dir)
			}
			for len(glob) > 0 {
				dir, glob = glob[len(glob)-1], glob[:len(glob)-1]
				if f, err = fs.Open(dir); err != nil {
					return
				}
				if fi, err = f.Readdir(0); err != nil {
					f.Close()
					return
				}
				f.Close()
				for _, fi := range fi {
					if fi.IsDir() && isnotvcs(fi.Name()) {
						glob = append(glob, filepath.Join(dir, fi.Name()))
					}
				}
				dirs[dir] = struct{}{}
			}
		default:
			if dir, err = lookup(fs, paths, pkg); err != nil {
				return
			}
			dirs[dir] = struct{}{}
		}
	}
	for dir := range dirs {
		if f, err = fs.Open(dir); err != nil {
			files = nil
			return
		}
		if fi, err = f.Readdir(0); err != nil {
			f.Close()
			files = nil
			return
		}
		f.Close()
		for _, fi := range fi {
			if !fi.IsDir() && istestfile(fi.Name()) {
				files = append(files, filepath.Join(dir, fi.Name()))
			}
		}
	}
	return
}

func (u Ungocheck) Rewrite(files []string) (tests []string, err error) {
	fs := u.fs()
	defer func() {
		if err != nil {
			for _, test := range tests {
				fs.Remove(test)
			}
			tests = nil
		}
	}()
	var n int
	for _, file := range files {
		test := filepath.Base(file)
		test = test[:len(test)-len("_test.go")] + "_ungocheck_test.go"
		test = filepath.Join(filepath.Dir(file), test)
		if n, err = u.rewriteSingle(file, test); err != nil {
			return
		}
		if n == 0 {
			fs.Remove(test)
		} else {
			tests = append(tests, test)
		}
	}
	return
}

var re = regexp.MustCompile(`func \(.*\) Test([\w\d_]+)\(([\w\d_]+) \*C\) {`)

const f = "func Test%s(%s *testing.T) { // ungocheck\n	%s.Parallel() // ungocheck\n"

func (u Ungocheck) rewriteSingle(file, test string) (n int, err error) {
	var (
		fs  = u.fs()
		src File
		dst File
	)
	if src, err = fs.Open(file); err != nil {
		return
	}
	defer src.Close()
	if dst, err = fs.OpenFile(test, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644); err != nil {
		return
	}
	defer dst.Close()
	// TODO(rjeczalik): reimplement with go/ast
	r, w, p := bufio.NewReader(src), bytes.NewBuffer([]byte("// +build ungocheck\n\n")), []byte{}
	for {
		if p, err = r.ReadBytes('\n'); len(p) == 0 {
			if err == io.EOF {
				_, err = io.Copy(dst, bytes.NewReader(w.Bytes()))
				return
			}
			if err != nil {
				return
			}
		}
		if bytes.HasPrefix(p, []byte("package ")) {
			w.Write(p)
			w.WriteString("\nimport \"testing\"\n")
			continue
		}
		if bytes.Contains(p, []byte("// +build")) && bytes.Contains(p, []byte("!ungocheck")) {
			continue
		}
		m := re.FindAllSubmatch(p, 1)
		if m != nil && len(m) == 1 && len(m[0]) == 3 {
			n += 1
			p = []byte(fmt.Sprintf(f, m[0][1], m[0][2], m[0][2]))
		} else if bytes.Contains(p, []byte("c *C")) {
			p = bytes.Replace(p, []byte("c *C"), []byte("c *testing.T"), 1)
		}

		w.Write(p)
	}
}

func (u Ungocheck) Run(args []string) (err error) {
	var (
		tests []string
		me    MultiError
	)
	defer func() {
		err = me.Consume(err).Consume(u.runTest(args, tests)).Err()
	}()
	s := u.Packages(args[1:])
	if s, err = u.Files(s); err != nil {
		return
	}
	tests, err = u.Rewrite(s)
	return
}
func (u Ungocheck) fs() (fs Filesystem) {
	fs = DefaultFilesystem
	if u.FS != nil {
		fs = u.FS
	}
	return
}

func (u Ungocheck) runTest(args, tests []string) error {
	var (
		me  MultiError
		arg = append([]string{"test", "-tags=ungocheck"}, args[1:]...)
	)
	fs := u.fs()
	out, err := exec.Command("go", arg...).CombinedOutput()
	me.Consume(err).Consume(io.Copy(os.Stdout, bytes.NewReader(out)))
	for _, test := range tests {
		me.Consume(fs.Remove(test))
	}
	return me
}
