package util

import (
	"fmt"
)

func Assert(cond bool, format string, args ...interface{}) {
	if cond == false {
		panic(fmt.Sprintf(format, args...))
	}
}
