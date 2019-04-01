package unittest

import (
	"io/ioutil"
	"os"
	"testing"
)

func WithTempFile(t *testing.T, fileName string, handler func(t *testing.T, f *os.File)) {
	f, err := ioutil.TempFile("", fileName)
	Assert(t, err == nil, "create fmt file failed %v", err)
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()
	handler(t, f)
}
