package logger

import (
	"eggdfs/svc/conf"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

const (
	envDebug      = "debug"
	envProduction = "prod"
)

var (
	zapLog *zap.Logger
)

func initLogger() {
	writeSyncer := getLogWriter()
	var cores []zapcore.Core

	switch conf.Config().Env {
	case envDebug:
		consoleCore := zapcore.NewCore(getConsoleEncoder(), zapcore.Lock(os.Stdout), zap.DebugLevel)
		cores = append(cores, consoleCore)
	case envProduction:
		fileCore := zapcore.NewCore(getEncoder(), writeSyncer, zapcore.InfoLevel)
		cores = append(cores, fileCore)
	}

	core := zapcore.NewTee(cores...)
	zapLog = zap.New(core, zap.AddCaller())
	zapLog.Info("日志服务启动...", zap.String("Env", conf.Config().Env))
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

func getConsoleEncoder() zapcore.Encoder {
	return zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
}

func getLogWriter() zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   conf.Config().LogDir,
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}
	return zapcore.AddSync(lumberJackLogger)
}

func Info(msg string, fields ...zap.Field) {
	zapLog.Info(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	zapLog.Debug(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	zapLog.Error(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	zapLog.Warn(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	zapLog.Panic(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	zapLog.Fatal(msg, fields...)
}

func init() {
	initLogger()
}
