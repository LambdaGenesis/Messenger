package main

import (
	"context"
	"html/template"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
	"strconv"
	// "fmt"

	"github.com/jackc/pgx/v4"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/gorilla/websocket"
)

type Message struct{
	Username string `json:"username"`
	Content string 	`json:"content"`
	Time string 	`json:"time"`
}

type Client struct {
	conn *websocket.Conn
	send chan Message
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
	broadcast  = make(chan Message, 100)
)

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error{
	return t.templates.ExecuteTemplate(w, name, data)
}

func main(){
	initDB()
	HandleRequests()
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer ws.Close()

	// Создаем клиента
	client := &Client{
		conn: ws,
		send: make(chan Message, 100),
	}

	// Регистрируем клиента
	registerClient(client)

	// Удаляем клиента при отключении
	defer unregisterClient(client)

	// Отправляем приветственное сообщение
	welcomeMsg := Message{
		Username: "System",
		Content:  "",
		Time:     time.Now().Format("15:04"),
	}
	client.send <- welcomeMsg

	// Читаем сообщения от клиента
	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Read error: %v", err)
			}
			break
		}
		// Добавляем время к сообщению
		msg.Time = time.Now().Format("15:04")
		log.Printf("Получено сообщение: %s", msg.Content)

		// Отправляем в канал широковещания
		broadcast <- msg
	}
}

// Регистрация клиента
func registerClient(client *Client) {
	clientsMux.Lock()
	defer clientsMux.Unlock()
	clients[client] = true
	log.Printf("Новый клиент подключен. Всего клиентов: %d", len(clients))

	// Запускаем горутину для отправки сообщений клиенту
	go client.writePump()
}

// Удаление клиента
func unregisterClient(client *Client) {
	clientsMux.Lock()
	defer clientsMux.Unlock()
	delete(clients, client)
	log.Printf("Клиент отключен. Осталось клиентов: %d", len(clients))

	// Отправляем уведомление о выходе
	leaveMsg := Message{
		Username: "System",
		Content:  "Пользователь вышел из чата",
		Time:     time.Now().Format("15:04"),
	}
	broadcast <- leaveMsg

	close(client.send)
}

// Отправка сообщений клиенту
func (c *Client) writePump() {
	for {
		msg, ok := <-c.send
		if !ok {
			// Канал закрыт
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
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
			case client.send <- msg:
				// Сообщение отправлено в канал клиента
			default:
				// Если канал полный, пропускаем сообщение
				log.Printf("Канал клиента переполнен, сообщение пропущено")
			}
		}
		clientsMux.Unlock()
	}
}

func HandleRequests(){
	e := echo.New()
	
	e.Use(middleware.Logger()) // миддлвейр 
	e.Use(middleware.Recover())

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete}, // CORS для разрешения с браузера 
	}))
	
	e.Static("/static", "static") // создаем для статических файлов CSS 

	templates, err := template.ParseFiles(
		"templates/footer.html",
	    "templates/header.html",
		"templates/contacts_page.html",
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
	e.Renderer = &Template{templates: templates}

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusPermanentRedirect, "/auth")
	})

	e.GET("/main", mainPage)
	e.GET("/home", homePage)
	e.GET("/auth", showAuthPage)
	e.POST("/auth/post", authPage)
	e.GET("/about", aboutPage)
	e.POST("/reg/post", regPage)
	e.GET("/reg", showRegPage)
	e.GET("/contacts", contactsPage)

	e.GET("/ws", func(c echo.Context) error {
		handleConnections(c.Response(), c.Request())
		return nil
	})

	go handleMessages()
	
	e.Logger.Fatal(e.Start("0.0.0.0:8080"))
	// e.Logger.Fatal(e.Start(":8080"))
}

func homePage(c echo.Context) error{
	return c.Render(http.StatusOK, "home_page", map[string]interface{}{
		"Title": "Home page",
	})
}

func mainPage(c echo.Context) error{
	return c.Render(http.StatusOK, "main_page", map[string]interface{}{
		"Title": "Main page",
	})
}

func aboutPage(c echo.Context) error{
	return c.Render(http.StatusOK, "about_page", map[string]interface{}{
		"Title": "About",
	})
}

func contactsPage(c echo.Context) error{
	return c.Render(http.StatusOK, "contacts_page", map[string]interface{}{
		"Title": "Chat",
	})
}

func showRegPage(c echo.Context) error{
	return c.Render(http.StatusOK, "reg_page", map[string]interface{}{
        "Title": "Registration",
        "Error": "", // пустая ошибка по приколу 
    })
}

func initDB(){
	// conn, err := pgx.Connect(context.Background(), "postgres://postgres:Roflan_2006@postgres:5432/data")
	conn, err := pgx.Connect(context.Background(), "postgres://postgres:Roflan_2006@localhost:5432/data")
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
		initDB() // рекурсия на проверку
		return
	}
}

func regPage(c echo.Context) error {
	if c.Request().Method != http.MethodPost{
		return c.Redirect(http.StatusFound, "/reg")
	}

	getUsernameReg := c.FormValue("usernameReg")
	getPasswordReg := c.FormValue("passwordReg")

	if _, err := strconv.Atoi(getPasswordReg); err != nil {
        return c.Render(http.StatusOK, "reg_page", map[string]interface{}{
            "Title": "Registration",
            "Error": "Password must contain only numbers",
        })
    }
	// проверка инфы с базы даннных 
	conn, err := pgx.Connect(context.Background(), "postgres://postgres:Roflan_2006@localhost:5432/data")
	// conn, err := pgx.Connect(context.Background(), "postgres://postgres:Roflan_2006@postgres:5432/data") // надо будет закинуть в gitignore и защитить от SQL инъекций, хз
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
		if getUsernameReg == username && getPasswordReg == stringPassword{

		// if getUsernameReg == username && getPasswordReg == password{
			data := struct{Error string}{Error: "Password or login is already exists"}
			return c.Render(http.StatusOK, "reg_page", data)
		}
	}// проверка инфы с таблиц базы данных
	writeSQL(getUsernameReg, getPasswordReg)
	return c.Render(http.StatusOK, "reg_page", nil)
}

func showAuthPage(c echo.Context) error {
	 return c.Render(http.StatusOK, "auth_page", map[string]interface{}{
        "Title": "Authorization",
        "Error": "", // added empty error for a template
    })
}

func authPage(c echo.Context) error{
	if c.Request().Method != http.MethodPost {
        return c.Redirect(http.StatusFound, "/auth")
    }

	getUsernameAuth := c.FormValue("username")
	getPasswordAuth := c.FormValue("password")

	// conn, err := pgx.Connect(context.Background(), "postgres://postgres:Roflan_2006@postgres:5432/data")
	conn, err := pgx.Connect(context.Background(), "postgres://postgres:Roflan_2006@localhost:5432/data")
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
		// if getUsernameAuth == username && getPasswordAuth == password{
			return c.Redirect(http.StatusFound, "/home")
		}
	}

	return c.Render(http.StatusOK, "auth_page", map[string]interface{}{
		"Title": "Authorization",
		"Error": "Wrong password or login",
	})
}

func writeSQL(username, password string) {
	conn, err := pgx.Connect(context.Background(), "postgres://postgres:Roflan_2006@localhost:5432/data")
	// conn, err := pgx.Connect(context.Background(), "postgres://postgres:Roflan_2006@postgres:5432/data") // надо будет закинуть в gitignore и защитить от SQL инъекций, хз
	if err != nil{
		log.Fatal(err)
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(), "INSERT INTO data_user (username, password) VALUES ($1, $2)", username, password) // нужно закинуть переменные, получаемые из строки в странице авторизации 
	if err != nil{
		log.Fatal(err)
	}
}