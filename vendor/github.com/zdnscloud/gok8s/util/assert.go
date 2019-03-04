package util

import (
	"fmt"
)

func Assert(condition bool, format string, v ...interface{}) {
	if condition == false {
		panic(fmt.Sprintf(format, v...))
	}
}
