package g53

import (
	"math"
	"path/filepath"
	"reflect"
	"runtime"
)

type TestingT interface {
	Logf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	FailNow()
}

func WireMatch(t TestingT, expectData []uint8, actualData []uint8) {
	if len(expectData) != len(actualData) {
		t.Errorf("want len [%v] but get [%v]", len(expectData), len(actualData))
	}

	minLen := int(math.Min(float64(len(expectData)), float64(len(actualData))))
	_, file, line, _ := runtime.Caller(1)
	for i := 0; i < minLen; i++ {
		if expectData[i] != actualData[i] {
			t.Errorf("at pos %d\n", i)
			t.Logf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n",
				filepath.Base(file), line, expectData[i:], actualData[i:])
			break
		}
	}
}

func NameEqToStr(t TestingT, n *Name, str string) {
	s, _ := NewName(str, true)
	if n.Equals(s) == false {
		_, file, line, _ := runtime.Caller(1)
		t.Logf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n",
			filepath.Base(file), line, str, n.String(true))
		t.FailNow()
	}
}

func Assert(t TestingT, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		t.Logf("\033[31m%s:%d: "+msg+"\033[39m\n\n",
			append([]interface{}{filepath.Base(file), line}, v...)...)
		t.FailNow()
	}
}

func Equal(t TestingT, act, exp interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		t.Logf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n",
			filepath.Base(file), line, exp, act)
		t.FailNow()
	}
}

func Nequal(t TestingT, act, exp interface{}) {
	if reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		t.Logf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n",
			filepath.Base(file), line, exp, act)
		t.FailNow()
	}
}
