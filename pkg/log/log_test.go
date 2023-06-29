package log

import (
	"testing"
)

func TestMyLog(t *testing.T) {
	s := "abc"
	//Debug("this is debug")
	//Debug("this is debug:%s\n", s)

	Info("this is info:%s\n", s)
}
