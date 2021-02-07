package server

import (
	"eggFs/conf"
	"eggFs/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
)

func config() *conf.GlobalConfig {
	return conf.Config()
}

func hello(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"msg": "hello eggFs!",
	})
}

func Start() {
	conf.ParseConfig(conf.ConfigFilePath)
	r := gin.Default()
	//gin.SetMode(gin.ReleaseMode)

	r.GET("/hello", hello)

	err := r.Run(config().Port)
	if err != nil {
		logger.Panic("web服务启动失败", zap.String("port", config().Port))
	}
	logger.Info("web服务启动成功", zap.String("port", config().Port))
}
