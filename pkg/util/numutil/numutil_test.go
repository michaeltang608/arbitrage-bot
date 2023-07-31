package numutil

import "testing"

func TestA(t *testing.T) {
	println(FormatInt(31))
	println(FormatByPrecision(3.1425, 3))
}
