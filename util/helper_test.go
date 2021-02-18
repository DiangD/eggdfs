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
	md5, err := GenFileMD5(f)
	if err != nil {
		panic(err)
	}
	fmt.Println(md5)
}

func TestGenUUIDFileName(t *testing.T) {
	for i := 0; i < 100; i++ {
		fmt.Println(GenUUIDFileName("jpg"))
	}
}
