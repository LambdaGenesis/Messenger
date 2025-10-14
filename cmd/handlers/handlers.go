package handlers

import (
	"log"
	"net/http"
	"html/template"
	"io"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"Messanger/internal/websocket"
	"Messanger/internal/database"
)

type Template struct{
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error{
	return t.templates.ExecuteTemplate(w, name, data) // Рендер шаблонов 
}

func HandleRequests(){
	e := echo.New()
	
	e.Use(middleware.Logger()) // Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete}, // CORS для разрешения с браузера 
	}))
	
	e.Static("/web/static", "web/static") // Создаем для статических файлов CSS (красивые штучки:))))

	templates, err := template.ParseFiles( // Обработка HTML-файлов (ну тупо страниц)
		"web/templates/footer.html",
	    "web/templates/header.html",
		"web/templates/contacts_page.html",
		"web/templates/channels_page.html",
		"web/templates/side_bar.html",
	    "web/templates/main_page.html",
	    "web/templates/auth_page.html",
	    "web/templates/home_page.html",
		"web/templates/about_page.html",
		"web/templates/reg_page.html",
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
	e.POST("/auth/post", database.AuthPage)
	e.GET("/channs", channelsPage)
	e.GET("/about", aboutPage)
	e.POST("/reg/post", database.RegPage)
	e.GET("/reg", showRegPage)
	e.GET("/chat", contactsPage)

	e.GET("/ws", func(c echo.Context) error {
		websocket.HandleConnections(c.Response(), c.Request())
		return nil
	})
	go websocket.HandleMessages()
	
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
// Функция, показывающая страницу авторизации
func showAuthPage(c echo.Context) error { 
	 return c.Render(http.StatusOK, "auth_page", map[string]interface{}{
        "Title": "Authorization",
        "Error": "", // Пустой шаблон, хз зачем по приколу ахахахах
    })
}
