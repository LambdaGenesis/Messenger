package main

import (
	"html/template"
	"net/http"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"io"
	"log"
)

type Template struct{
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error{
	return t.templates.ExecuteTemplate(w, name, data)
}

func main(){
	HandleRequests()
}

func HandleRequests(){
	e := echo.New()
	
	e.Use(middleware.Logger()) // creating a middleware for a programm
	e.Use(middleware.Recover())
	
	e.Static("/static", "static") // creating static files 

	templates, err := template.ParseFiles(
		"templates/footer.html",
	    "templates/header.html",
	    "templates/main_page.html",
	    "templates/auth_page.html",
	    "templates/home_page.html",
	)
	if err != nil {
		log.Fatalf("Ошибка загрузки шаблонов: %v", err)
	}
	e.Renderer = &Template{templates: templates}
	
	e.GET("/", mainPage)
	e.GET("/home", homePage)
	e.GET("/auth", showAuthPage)
	e.POST("/auth/post", authPage)
	
	e.Logger.Fatal(e.Start(":8080"))
}

func homePage(c echo.Context) error{
	return c.Render(http.StatusOK, "home_page.html", map[string]interface{}{
		"Title": "Домашняя страница",
	})
}

func mainPage(c echo.Context) error{
	return c.Render(http.StatusOK, "main_page.html", map[string]interface{}{
		"Title": "Главная страница",
	})
}

func showAuthPage(c echo.Context) error {
	 return c.Render(http.StatusOK, "auth_page.html", map[string]interface{}{
        "Title": "Авторизация",
        "Error": "", // Добавляем пустую ошибку для шаблона
    })
}

func authPage(c echo.Context) error{
	if c.Request().Method != http.MethodPost {
        return c.Redirect(http.StatusFound, "/auth")
    }

	username := c.FormValue("username")
	password := c.FormValue("password")

	if username == "aaa" && password == "aaa"{
		return c.Redirect(http.StatusFound, "/home")
	}
	data := struct{ Error string }{Error: "Неверный логин или пароль"}
	return c.Render(http.StatusOK, "auth_page.html", data)
}
