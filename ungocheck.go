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

func (u Ungocheck) Files(_ []string) (files []string, err error) {
	// TODO(rjeczalik): implement directory/packages globbing
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	fs := u.fs()
	dir, err := fs.Open(wd)
	if err != nil {
		return
	}
	defer dir.Close()
	fi, err := dir.Readdir(0)
	if err != nil {
		return
	}
	for _, fi := range fi {
		if strings.HasSuffix(fi.Name(), "_test.go") && !strings.HasSuffix(fi.Name(), "_ungocheck_test.go") {
			files = append(files, filepath.Join(wd, fi.Name()))
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
		}
		w.Write(p)
	}
	_ = w
	return
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
