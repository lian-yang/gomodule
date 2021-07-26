package webscoket


type Hub struct {
	// 注册客户端集合
	clients map[*Client]bool

	// 广播消息给所有客户端
	broadcast chan []byte

	// 注册客户端
	register chan *Client

	// 注销客户端
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.message)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.message <- message:
				default:
					close(client.message)
					delete(h.clients, client)
				}
			}
		}
	}
}