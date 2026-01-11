package fileUtils

import "testing"

func Test_fileName(t *testing.T) {
	name := GetRandomFileName("1.png")
	t.Log(name)
}
