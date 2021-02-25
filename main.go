package main

import (
	"eggdfs/common/logo"
	"eggdfs/svc"
)

func main() {
	logo.PrintLogo()
	svc.Start()
}
