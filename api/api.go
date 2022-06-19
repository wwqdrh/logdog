package api

import (
	"io/fs"
	"net/http"
	"wwqdrh/logdog/logx"

	"github.com/gin-gonic/gin"
)

var Tracing *logx.LocalTracing

func NewEngine(logPath string) (*gin.Engine, error) {
	if val, err := logx.NewLocaltracing(logPath); err != nil {
		return nil, err
	} else {
		Tracing = val
	}
	engine := gin.Default()
	engine.GET("/", viewIndex)
	engine.GET("/health", health)
	engine.GET("/log/list", logList)
	engine.GET("/log/data", logData)

	v, _ := fs.Sub(views, "views")
	ExecuteStatic(engine, "/static", http.FS(v))
	return engine, nil
}

// 健康检查
func health(ctx *gin.Context) {
	ctx.String(200, "ok")
}

// 当前的日志文件列表
func logList(ctx *gin.Context) {
	ctx.String(200, "ok")
}
