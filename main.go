package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/joho/godotenv"
)

type Message struct{
	Username string `json:"username"`
	Content string 	`json:"content"`
	Time string 	`json:"time"`
}

type Client struct {
	conn *websocket.Conn
	send chan Message
	stopPing chan bool
}

type Template struct{
	templates *template.Template
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


// func makeChan(c echo.Context){
// 	// btnMakeChan := c.FormValue("btnMakeChan")

// }	

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error{
	return t.templates.ExecuteTemplate(w, name, data) // Рендер шаблонов 
}

func main(){
	err := godotenv.Load()
	if err != nil{
		log.Println("Can't connect to .env file!")
	}
	initDB() // Проверка на базу данных
	HandleRequests() 
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer ws.Close()

	ws.SetReadDeadline(time.Now().Add(180 * time.Second)) // было 60 сек
    ws.SetPongHandler(func(string) error { 
        ws.SetReadDeadline(time.Now().Add(180 * time.Second)) // было 60 сек
        return nil 
    })	
	client := &Client{ // Создаем клиента
		conn: ws,
		send: make(chan Message, 100),
		stopPing: make(chan bool),
	}
	go pingClient(client)
	
	registerClient(client) // Регистрируем клиента
	defer unregisterClient(client) // Удаляем клиента при отключении

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

func pingClient(client *Client){
	ticker := time.NewTicker(30 * time.Second) // Ping каждые 30 секунд
    defer ticker.Stop()
    
    for {
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
func registerClient(client *Client) { 
	clientsMux.Lock()
	defer clientsMux.Unlock()
	clients[client] = true
	newClientMessage := Message{ // Уведомление о приходе нового клиента
		Username: "System",
		Content: "Пользователь зашел в чат",
		Time: 	time.Now().Format("15:04"),

		// ID: true,
	}	
	broadcast <- newClientMessage // Отправка сообщения в общий канал
	log.Printf("Новый клиент подключен. Всего клиентов: %d", len(clients))
	go client.writePump() // Запуск горутины для отправки сообщений клиенту
}
// Удаление клиента
func unregisterClient(client *Client) { 
	clientsMux.Lock()
	defer clientsMux.Unlock()

	close(client.stopPing)

	delete(clients, client)
	log.Printf("Клиент отключен. Осталось клиентов: %d", len(clients))

	leaveMsg := Message{ // Отправляем уведомление о выходе клиента
		Username: "System",
		Content:  "Пользователь вышел из чата",
		Time:     time.Now().Format("15:04"),

		// ID: true,
	}
	broadcast <- leaveMsg // Отправка сообщения в общий канал
	close(client.send)
}
// Отправка сообщений клиенту
func (c *Client) writePump() { 
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
func handleMessages() {  
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

func HandleRequests(){
	e := echo.New()
	
	e.Use(middleware.Logger()) // Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete}, // CORS для разрешения с браузера 
	}))
	
	e.Static("/static", "static") // Создаем для статических файлов CSS (красивые штучки:))))

	templates, err := template.ParseFiles( // Обработка HTML-файлов (ну тупо страниц)
		"templates/footer.html",
	    "templates/header.html",
		"templates/contacts_page.html",
		"templates/channels_page.html",
		"templates/side_bar.html",
	    "templates/main_page.html",
	    "templates/auth_page.html",
	    "templates/home_page.html",
		"templates/about_page.html",
		"templates/reg_page.html",
	)
	if err != nil {
		log.Fatalf("Ошибка загрузки шаблонов: %v", err)
	}
	e.Renderer = &Template{templates: templates} // Рендер шаблонов 

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusPermanentRedirect, "/auth")
	})

	e.GET("/main", mainPage)
	e.GET("/home", homePage)
	e.GET("/auth", showAuthPage)
	e.POST("/auth/post", authPage)
	e.GET("/channs", channelsPage)
	e.GET("/about", aboutPage)
	e.POST("/reg/post", regPage)
	e.GET("/reg", showRegPage)
	e.GET("/chat", contactsPage)

	e.GET("/ws", func(c echo.Context) error {
		handleConnections(c.Response(), c.Request())
		return nil
	})
	go handleMessages()
	
	e.Logger.Fatal(e.Start("0.0.0.0:8080")) // Хост для показа всем интерфейсам
}
// Домашняя страница
func homePage(c echo.Context) error{ 
	return c.Render(http.StatusOK, "home_page", map[string]interface{}{
		"Title": "Home page",
	})
}
// Главная страница
func mainPage(c echo.Context) error{ 
	return c.Render(http.StatusOK, "main_page", map[string]interface{}{
		"Title": "Main page",
	})
}
// Страница о самом Месседжере
func aboutPage(c echo.Context) error{ 
	return c.Render(http.StatusOK, "about_page", map[string]interface{}{
		"Title": "About",
	})
}
func channelsPage(c echo.Context) error{
	return c.Render(http.StatusOK, "channels_page", map[string]interface{}{
		"Title": "Contacts",
	})
}
// Страница чата
func contactsPage(c echo.Context) error{ 
	return c.Render(http.StatusOK, "contacts_page", map[string]interface{}{
		"Title": "Chat",
	})
}
// Функция, показывающая страницу регистрации
func showRegPage(c echo.Context) error{ 
	return c.Render(http.StatusOK, "reg_page", map[string]interface{}{
        "Title": "Registration",
        "Error": "", // пустая ошибка по приколу 
    })
}
// Функция для регистрации в Мессенджере
func regPage(c echo.Context) error { 	
	if c.Request().Method != http.MethodPost{
		return c.Redirect(http.StatusFound, "/reg")
	}
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", 
		os.Getenv("USER"),
		os.Getenv("PASSWORD"),
		os.Getenv("HOST"),
		os.Getenv("PORT"),
		os.Getenv("DB"),
	)
	getUsernameReg := c.FormValue("usernameReg")
	getPasswordReg := c.FormValue("passwordReg")
	if _, err := strconv.Atoi(getPasswordReg); err != nil {
        return c.Render(http.StatusOK, "reg_page", map[string]interface{}{
            "Title": "Registration",
            "Error": "Password must contain only numbers",
        })
    }
	// Проверка инфы с базы даннных 
	
	conn, err := pgx.Connect(context.Background(), connStr)
	//conn, err := pgx.Connect(context.Background(), "postgres://postgres:Roflan_2006@postgres:5432/data") // надо будет закинуть в gitignore и защитить от SQL инъекций, хз
	if err != nil{
		log.Printf("Error: %v",err)
		return c.Render(http.StatusOK, "auth_page", map[string]interface{}{
			"Title": "Authorization",
        	"Error": "Database connection error",
		})
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), "SELECT username, password FROM data_user")
	if err != nil{
		log.Fatal(err)
	}
	defer rows.Close()
	var (
		username string
		password int
	)
	for rows.Next(){
		err := rows.Scan(&username, &password)
		if err != nil{
			log.Fatal(err)
		}
		stringPassword := strconv.Itoa(password)
		if getUsernameReg == username || getPasswordReg == stringPassword{
			data := struct{Error string}{Error: "Password or login is already exists"}
			return c.Render(http.StatusOK, "reg_page", data)
		}
	}// проверка инфы с таблиц базы данных
	writeSQL(getUsernameReg, getPasswordReg)
	return c.Render(http.StatusOK, "reg_page", nil)
}
// Функция, показывающая страницу авторизации
func showAuthPage(c echo.Context) error { 
	 return c.Render(http.StatusOK, "auth_page", map[string]interface{}{
        "Title": "Authorization",
        "Error": "", // Пустой шаблон, хз зачем по приколу ахахахах
    })
}
// Функция для авторизации в Мессенджере
func authPage(c echo.Context) error{ 
	if c.Request().Method != http.MethodPost {
        return c.Redirect(http.StatusFound, "/auth")
    }
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", 
		os.Getenv("USER"),
		os.Getenv("PASSWORD"),
		os.Getenv("HOST"),
		os.Getenv("PORT"),
		os.Getenv("DB"),
	)
	getUsernameAuth := c.FormValue("username")
	getPasswordAuth := c.FormValue("password")

	//conn, err := pgx.Connect(context.Background(), "postgres://postgres:Roflan_2006@postgres:5432/data")
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil{
		log.Printf("Error: %v",err)
		return c.Render(http.StatusOK, "auth_page", map[string]interface{}{
			"Title": "Authorization",
        	"Error": "Database connection error",
		})
	}
	defer conn.Close(context.Background())
	
	rows, err := conn.Query(context.Background(), "SELECT username, password FROM data_user")
	if err != nil{
		log.Fatal(err)
	}
	defer rows.Close()

	var (
		username string
		password int
	)
	
	for rows.Next(){
		err := rows.Scan(&username, &password)
		if err != nil{
			return c.Render(http.StatusOK, "auth_page", map[string]interface{}{
				"Title": "Authorization",
        		"Error": "Wrong password or login",
			})
		}
		stringPassword := strconv.Itoa(password)
		if getUsernameAuth == username && getPasswordAuth == stringPassword{
			return c.Redirect(http.StatusFound, "/home")
		}
	}
	return c.Render(http.StatusOK, "auth_page", map[string]interface{}{
		"Title": "Authorization",
		"Error": "Wrong password or login",
	})
}
// Проверка на наличие базы данных, если ее нет, он ее создает
func initDB(){
	//conn, err := pgx.Connect(context.Background(), "postgres://postgres:Roflan_2006@postgres:5432/data")
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", 
		os.Getenv("USER"),
		os.Getenv("PASSWORD"),
		os.Getenv("HOST"),
		os.Getenv("PORT"),
		os.Getenv("DB"),
	)
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil{
		log.Fatalf("%v",err)
	}
	_, err = conn.Exec(context.Background(), `
        CREATE TABLE IF NOT EXISTS data_user (
            username VARCHAR(50) UNIQUE NOT NULL,
            password INT NOT NULL
        )
    `)
	if err != nil{
		time.Sleep(2 * time.Second)
		initDB() // Рекурсия на проверку 
		return
	}
}
// Запись информации о клиенте в базу данных
func writeSQL(username, password string) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", 
		os.Getenv("USER"),
		os.Getenv("PASSWORD"),
		os.Getenv("HOST"),
		os.Getenv("PORT"),
		os.Getenv("DB"),
	)
	conn, err := pgx.Connect(context.Background(), connStr)
	//conn, err := pgx.Connect(context.Background(), "postgres://postgres:Roflan_2006@postgres:5432/data") // Надо будет закинуть в gitignore и защитить от SQL инъекций, хз не придумал
	if err != nil{
		log.Fatal(err)
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(), "INSERT INTO data_user (username, password) VALUES ($1, $2)", username, password) // Нужно закинуть переменные, получаемые из строки в странице авторизации 
	if err != nil{
		log.Fatal(err)
	}
}