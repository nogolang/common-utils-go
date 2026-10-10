package fileUtils

import (
	"testing"

	"github.com/nogolang/common-utils-go/uploadUtils"
)

func Test_fileName(t *testing.T) {
	name := uploadUtils.GetRandomFileName("image", "png")
	if name == "" {
		t.Fatal("random file name is empty")
	}
	t.Log(name)
}
