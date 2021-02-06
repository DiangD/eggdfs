package server

import (
	"eggFs/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Start() {
	router := gin.New()
	router.Use(gin.Recovery())
	//gin.SetMode(gin.ReleaseMode)

	err := router.Run(":8080")
	if err != nil {
		logger.Panic("web服务启动失败", zap.String("port", "8080"))
	}
	logger.Info("web服务启动成功", zap.String("port", "8080"))
}
