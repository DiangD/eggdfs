package conf

import (
	"fmt"
	"testing"
)

func TestParseConfig(t *testing.T) {
	parseConfig()
	fmt.Printf("%+v", Config())
}
