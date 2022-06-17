package api

import (
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
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

// websocket read healper func
func WsRead(conn *websocket.Conn, done chan bool) {
	defer func() {
		conn.Close()
		select {
		case _, ok := <-done:
			if !ok {
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
			if !ok {
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
