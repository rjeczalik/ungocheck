// +build !ungocheck

package example

import (
	. "gopkg.in/check.v1"
)

type BarSuite []string

var bs = BarSuite{
	"eins",
	"zwei",
	"drei",
}

var _ = Suite(bs)

func (bs BarSuite) TestBarThis(c *C) {
	cases := []string{
		"einsbar",
		"zweibar",
		"dreibar",
	}
	for i, cas := range cases {
		if s := Bar(bs[i]); s != cas {
			c.Errorf("want Bar(%q)=%q; got %q (i=%d)", bs[i], cas, s, i)
		}
	}
}

func (bs BarSuite) TestBarThat(c *C) {
	cases := []string{
		"einsbarbar",
		"zweibarbar",
		"dreibarbar",
	}
	for i, cas := range cases {
		if s := Bar(Bar(bs[i])); s != cas {
			c.Errorf("want Bar(Bar(%q))=%q; got %q (i=%d)", bs[i], cas, s, i)
		}
	}
}
