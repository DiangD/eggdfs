package svc

import (
	"eggdfs/common"
	"eggdfs/svc/conf"
)

func config() *conf.GlobalConfig {
	return conf.Config()
}

func Start() {
	if config() == nil {
		conf.InitGlobalConfig()
	}
	c := config()
	switch c.DeployType {
	case common.DeployTypeStorages:
		NewStorage().Start()
	case common.DeployTypeTracker:
		NewTracker().Start()
	}
}
