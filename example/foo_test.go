// +build !ungocheck

package example

import (
	. "gopkg.in/check.v1"
)

type FooSuite []string

var fs = FooSuite{
	"one",
	"two",
	"three",
}

var _ = Suite(fs)

func (fs FooSuite) TestFooThis(c *C) {
	cases := []string{
		"onefoo",
		"twofoo",
		"threefoo",
	}
	for i, cas := range cases {
		if s := Foo(fs[i]); s != cas {
			c.Errorf("want Foo(%q)=%q; got %q (i=%d)", fs[i], cas, s, i)
		}
	}
}

func (fs FooSuite) TestFooThat(c *C) {
	cases := []string{
		"onefoofoo",
		"twofoofoo",
		"threefoofoo",
	}
	for i, cas := range cases {
		if s := Foo(Foo(fs[i])); s != cas {
			c.Errorf("want Foo(Foo(%q))=%q; got %q (i=%d)", fs[i], cas, s, i)
		}
	}
}
