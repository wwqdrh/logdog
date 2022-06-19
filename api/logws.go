package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/wwqdrh/logger"
)

var (
	newline = []byte{'\n'}
	// space   = []byte{' '}
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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

// websocket read healper func
func WsRead(conn *websocket.Conn, done chan bool) {
	defer func() {
		conn.Close()
		select {
		case _, ok := <-done:
			if ok {
				close(done)
			}
		default:
			close(done)
		}
		fmt.Println("退出wsread")
	}()
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
	}
}

func WsWrite(conn *websocket.Conn, send chan string, done chan bool) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		select {
		case _, ok := <-done:
			if ok {
				close(done)
			}
		default:
			close(done)
		}
		conn.Close()
		fmt.Println("退出wswrite")
	}()
	for {
		select {
		case message, ok := <-send:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write([]byte(message))

			// Add queued chat messages to the current websocket message.
			n := len(send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write([]byte(<-send))
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}
