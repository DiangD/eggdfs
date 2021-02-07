package server

import (
	"eggFs/conf"
	"eggFs/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func config() *conf.GlobalConfig {
	return conf.Config()
}

func Start() {
	conf.ParseConfig(conf.ConfigFilePath)
	router := gin.New()
	router.Use(gin.Recovery())
	//gin.SetMode(gin.ReleaseMode)

	err := router.Run(config().Port)
	if err != nil {
		logger.Panic("web服务启动失败", zap.String("port", "8080"))
	}
	logger.Info("web服务启动成功", zap.String("port", "8080"))
}
