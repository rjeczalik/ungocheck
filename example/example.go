package example

import "time"

func Foo(s string) string {
	time.Sleep(time.Second / 10)
	return s + "foo"
}

func Bar(s string) string {
	time.Sleep(time.Second / 10)
	return s + "bar"
}
