package logx

import (
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/hpcloud/tail"
	"github.com/wwqdrh/logger"
	"go.uber.org/zap"
)

var (
	// 默认日志文件
	baseLog = "base.log"
)

var tailHandler = map[string]*tailInfo{} // 文件名与channel的映射

type tailInfo struct {
	cmd *tail.Tail // 获取最新的日志数据
	chs []connNode // 多个连接进行复用
}

type connNode struct {
	ch   chan string
	done chan bool
}

type LocalTracing struct {
	*zap.Logger

	LogDir string
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func NewLocaltracing(logDir string) (*LocalTracing, error) {
	if ok, _ := PathExists(logDir); !ok {
		_ = os.MkdirAll(logDir, os.ModePerm)
	}

	handler := &LocalTracing{
		Logger: logger.NewLogger(logger.WithColor(true), logger.WithLogPath(path.Join(logDir, baseLog))),
		LogDir: logDir,
	}
	go func() {
		c1 := make(chan os.Signal, 1)
		signal.Notify(c1, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		// 接受信号同步日志
		handler.Sync()
	}()
	return handler, nil
}

// 每一个要读取的file可能由多个ws连接， 要复用则包装tails，并加上一系列channel
func (l *LocalTracing) TailLog(fileName string, done chan bool) chan string {
	cur := make(chan string, 1000)
	if val, ok := tailHandler[fileName]; ok {
		val.chs = append(val.chs, connNode{
			ch:   cur,
			done: done,
		})
		return cur
	}

	config := tail.Config{
		ReOpen:    true,                                 // 重新打开
		Follow:    true,                                 // 是否跟随
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2}, // 从文件的哪个地方开始读
		MustExist: false,                                // 文件不存在不报错
		Poll:      true,
	}
	tails, err := tail.TailFile(fileName, config)
	if err != nil {
		logger.DefaultLogger.Error("tail file failed, err:" + err.Error())
		return nil
	}
	handler := &tailInfo{
		cmd: tails,
		chs: []connNode{
			{
				ch:   cur,
				done: done,
			},
		},
	}
	tailHandler[fileName] = handler
	go func() {
		var (
			line *tail.Line
			ok   bool
		)
		for {
			line, ok = <-tails.Lines
			if !ok {
				logger.DefaultLogger.Info("tail file close reopen, filename:" + tails.Filename)
				time.Sleep(time.Second)
				continue
			}

			// 为所有的channel发送
			chs := handler.chs[:0]
			for _, item := range handler.chs {
				select {
				case <-item.done:
					continue
				case item.ch <- line.Text:
					chs = append(chs, item)
				default:
				}
			}
			handler.chs = chs // 剔除已经关闭的
		}
	}()
	return cur
}
