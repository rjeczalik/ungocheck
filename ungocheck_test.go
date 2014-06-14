package ungocheck

import (
	"reflect"
	"testing"
)

var u = New()

func TestPackages(t *testing.T) {
	cases := []struct {
		arg []string
		pkg []string
	}{{
		[]string{},
		[]string{"."},
	}, {
		[]string{"."},
		[]string{"."},
	}, {
		[]string{"./.."},
		[]string{"./.."},
	}, {
		[]string{"-v", "-race"},
		[]string{"."},
	}, {
		[]string{"-v", "-race", "pkg"},
		[]string{"pkg"},
	}, {
		[]string{"-v", "-race", "-test.parallel=4", "user/pkg1", "user/pkg2"},
		[]string{"user/pkg1", "user/pkg2"},
	}}
	for i, cas := range cases {
		if pkg := u.Packages(cas.arg); !reflect.DeepEqual(pkg, cas.pkg) {
			t.Errorf("want u.Packages=%v; got %v (i=%d)", cas.pkg, pkg, i)
		}
	}
}

func TestFiles(t *testing.T) {
	t.Skip("TODO(rjeczalik)")
}

func TestRewrite(t *testing.T) {
	t.Skip("TODO(rjeczalik)")
}
