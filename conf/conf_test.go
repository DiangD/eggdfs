package conf

import (
	"fmt"
	"testing"
)

func TestParseConfig(t *testing.T) {
	ParseConfig()
	fmt.Printf("%+v", Config())
}
