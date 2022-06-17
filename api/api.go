package api

import (
	"embed"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"wwqdrh/logdog/logx"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/wwqdrh/logger"
)

//go:embed views
var views embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

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
	return engine, nil
}

// 首页视图
func viewIndex(ctx *gin.Context) {
	logfile := ctx.Query("file")

	f, err := views.Open("index.html")
	if err != nil {
		ctx.String(500, err.Error())
		return
	}

	body, err := ioutil.ReadAll(f)
	if err != nil {
		ctx.String(500, err.Error())
		return
	}

	if err := ExecuteTemplate(ctx.Writer, "index", body, map[string]interface{}{"PageTitle": "实时日志", "LogFile": logfile}); err != nil {
		ctx.String(500, err.Error())
		return
	}
}

// 健康检查
func health(ctx *gin.Context) {
	ctx.String(200, "ok")
}

// 当前的日志文件列表
func logList(ctx *gin.Context) {
	ctx.String(200, "ok")
}

func logData(ctx *gin.Context) {
	ws, err := upgrader.Upgrade(ctx.Writer, ctx.Request, ctx.Writer.Header())
	if err != nil {
		ctx.String(500, "upgrade error")
		return
	}

	// check log is exist?
	file := path.Join(Tracing.LogDir, ctx.Query("file"))
	if _, err := os.Stat(file); os.IsNotExist(err) {
		ws.WriteMessage(websocket.TextMessage, []byte("log error: 日志文件不存在"))
		if e := ws.Close(); e != nil {
			logger.DefaultLogger.Error(e.Error())
		}
		return
	}

	close := make(chan bool)
	go WsRead(ws, close)
	go WsWrite(ws, Tracing.TailLog(file, close), close)
}
