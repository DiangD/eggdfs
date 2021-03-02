package util

import (
	"fmt"
	"os"
	"testing"
)

func TestGenFileMD5(t *testing.T) {
	f, err := os.Open("../meta/test.txt")
	defer func() {
		_ = f.Close()
	}()
	md5, err := GenMD5(f)
	if err != nil {
		panic(err)
	}
	fmt.Println(md5)
}
