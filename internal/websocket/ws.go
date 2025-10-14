package websocket

import (
	"time"
	"sync"
	"net/http"
	"log"
	"github.com/gorilla/websocket"
)

type Message struct{ // Определение структуры сообщения
	Username string `json:"username"`
	Content string 	`json:"content"`
	Time string 	`json:"time"`
}

type Client struct { // Определяем структуру клиента 
	conn *websocket.Conn
	send chan Message
	stopPing chan bool
}

var (
	upgrader = websocket.Upgrader{
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool{
		return true
	},}
	clients    = make(map[*Client]bool)
	clientsMux = &sync.Mutex{}
	broadcast  = make(chan Message, 100) // Общий канал 
)

func HandleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer ws.Close()

	ws.SetReadDeadline(time.Now().Add(180 * time.Second)) 
    ws.SetPongHandler(func(string) error { 
        ws.SetReadDeadline(time.Now().Add(180 * time.Second)) 
        return nil 
    })	
	client := &Client{ // Создаем клиента
		conn: ws,
		send: make(chan Message, 100),
		stopPing: make(chan bool),
	}
	go PingClient(client)
	
	RegisterClient(client) // Регистрируем клиента
	defer UnregisterClient(client) // Удаляем клиента при отключении

	for { // Читаем сообщения от клиента
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Read error: %v", err)
			}
			break
		}
		msg.Time = time.Now().Format("15:04") // Добавляем время к сообщению
		log.Printf("Получено сообщение: %s", msg.Content)

		broadcast <- msg // Отправляем в канал широковещания
	}
}

func PingClient(client *Client){
	ticker := time.NewTicker(30 * time.Second) 
    defer ticker.Stop()
    
    for { // бесконечный цикл
        select {
        case <-ticker.C:
            client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                log.Printf("Ping error: %v", err)
                return
            }
        case <-client.stopPing: // Остановить если канал закрыт
            return
        }
    }
}
// Регистрация клиента
func RegisterClient(client *Client) { 
	clientsMux.Lock()
	defer clientsMux.Unlock()
	clients[client] = true
	newClientMessage := Message{ // Уведомление о приходе нового клиента
		Username: "System",
		Content: "Пользователь зашел в чат",
		Time: 	time.Now().Format("15:04"),
	}	
	broadcast <- newClientMessage // Отправка сообщения в общий канал
	log.Printf("Новый клиент подключен. Всего клиентов: %d", len(clients))
	go client.WritePump() // Запуск горутины для отправки сообщений клиенту
}
// Удаление клиента
func UnregisterClient(client *Client) { 
	clientsMux.Lock()
	defer clientsMux.Unlock()

	close(client.stopPing)

	delete(clients, client)
	log.Printf("Клиент отключен. Осталось клиентов: %d", len(clients))

	leaveMsg := Message{ // Отправляем уведомление о выходе клиента
		Username: "System",
		Content:  "Пользователь вышел из чата",
		Time:     time.Now().Format("15:04"),
	}
	broadcast <- leaveMsg // Отправка сообщения в общий канал
	close(client.send)
}
// Отправка сообщений клиенту
func (c *Client) WritePump() { 
	for {
		msg, ok := <-c.send
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})  // Канал закрыт
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err := c.conn.WriteJSON(msg)
			if err != nil {
				log.Printf("Write error: %v", err)
				return
			}
	}
}
// Обработчик широковещательных сообщений
func HandleMessages() {  
	for {
		msg := <-broadcast
		clientsMux.Lock()
		for client := range clients {
			select {
				case client.send <- msg:  // Сообщение отправлено в канал клиента
				default:
					log.Printf("Канал клиента переполнен, сообщение пропущено")
			}
		}
		clientsMux.Unlock()
	}
}