package logo

import (
	"eggdfs/common"
	"fmt"
)

var Logo = `	
	______________________  ___________
   / ____/ ____/ ____/ __ \/ ____/ ___/
  / __/ / / __/ / __/ / / / /_   \__ \ 	EggDFS::v` + common.VERSION + `
 / /___/ /_/ / /_/ / /_/ / __/  ___/ / 	A distribute filesystem.
/_____/\____/\____/_____/_/    /____/  	https://github.com/DiangD/eggdfs
`

func PrintLogo() {
	fmt.Print(Logo)
}
