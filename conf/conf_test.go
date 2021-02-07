package conf

import (
	"fmt"
	"testing"
)

func TestParseConfig(t *testing.T) {
	path := "../config.json"
	ParseConfig(path)
	fmt.Printf("%+v", Config())
}
