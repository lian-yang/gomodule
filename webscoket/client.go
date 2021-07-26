package webscoket

import (
	"bytes"
	ws "github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

const (
	// 写等待时间
	writeWait = 10 * time.Second

	// pong等待时间
	pongWait = 60 * time.Second

	// 发送ping到客户端，必须小于pongWait
	pingPeriod = (pongWait * 9) / 10

	// 限制最大消息内容大小
	maxMessageSize = 1024
)

type Client struct {
	hub *Hub

	// 当前客户端链接.
	conn *ws.Conn

	// 消息缓冲通道
	message chan []byte
}

func (c *Client) read()  {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if ws.IsUnexpectedCloseError(err, ws.CloseGoingAway, ws.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, []byte{'\n'}, []byte{' '}, -1))
		c.hub.broadcast <- message
	}
}

func (c *Client) write()  {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.message:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// 发送关闭消息
				c.conn.WriteMessage(ws.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(ws.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 向当前客户端发送消息缓存队列的消息
			n := len(c.message)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.message)
			}

			if err = w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(ws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 解决跨域问题
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WsHandle 升级成websocket请求
func WsHandle(hub *Hub, w http.ResponseWriter, r *http.Request) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	client := &Client{hub: hub, conn: conn, message: make(chan []byte, 512)}
	client.hub.register <- client

	go client.read()
	go client.write()

	return nil
}