package api

import (
	"embed"
	"html/template"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wwqdrh/logger"
)

//go:embed views
var views embed.FS

// bindatatemplate 方法
func ExecuteTemplate(wr io.Writer, name string, body []byte, data interface{}) error {
	tmpl := template.New(name)

	newTmpl, err := tmpl.Parse(string(body))
	if err != nil {
		return err
	}
	return newTmpl.Execute(wr, data)
}

func ExecuteStatic(engine *gin.Engine, prefix string, filepath http.FileSystem) {
	f := http.FileServer(filepath)
	engine.HEAD(path.Join(prefix, "/*filepath"), func(ctx *gin.Context) {
		ctx.Request.URL.Path = strings.TrimPrefix(ctx.Request.URL.Path, prefix)
		f.ServeHTTP(ctx.Writer, ctx.Request)
	})
	engine.GET(path.Join(prefix, "/*filepath"), func(ctx *gin.Context) {
		ctx.Request.URL.Path = strings.TrimPrefix(ctx.Request.URL.Path, prefix)
		f.ServeHTTP(ctx.Writer, ctx.Request)
	})
}

// 首页视图
func viewIndex(ctx *gin.Context) {
	logfile := ctx.Query("file")

	f, err := fs.Sub(views, "views")
	if err != nil {
		logger.DefaultLogger.Error(err.Error())
		ctx.String(404, "not found")
		return
	}

	tmplF, err := f.Open("index.html")
	if err != nil {
		ctx.String(404, "not found")
		return
	}

	body, err := ioutil.ReadAll(tmplF)
	if err != nil {
		ctx.String(500, err.Error())
		return
	}

	if err := ExecuteTemplate(ctx.Writer, "index", body, map[string]interface{}{"PageTitle": "实时日志", "LogFile": logfile}); err != nil {
		ctx.String(500, err.Error())
		return
	}
}
