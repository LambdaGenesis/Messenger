package main

import (
	"context"
	"html/template"
	"io"
	"log"
	"net/http"

	"github.com/jackc/pgx/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
		"templates/side_bar.html",
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
	return c.Render(http.StatusOK, "home_page", map[string]interface{}{
		"Title": "Домашняя страница",
	})
}

func mainPage(c echo.Context) error{
	return c.Render(http.StatusOK, "main_page", map[string]interface{}{
		"Title": "Главная страница",
	})
}

func showAuthPage(c echo.Context) error {
	 return c.Render(http.StatusOK, "auth_page", map[string]interface{}{
        "Title": "Авторизация",
        "Error": "", // added empty error for a template
    })
}

func authPage(c echo.Context) error{
	if c.Request().Method != http.MethodPost {
        return c.Redirect(http.StatusFound, "/auth")
    }

	username := c.FormValue("username")
	password := c.FormValue("password")

	// databaseSQL(username, password)

	if username == "aaa" && password == "aaa"{
		return c.Redirect(http.StatusFound, "/home")
	}
	data := struct{ Error string }{Error: "Неверный логин или пароль"}
	return c.Render(http.StatusOK, "auth_page", data)
}

func databaseSQL(username string, password string){
	conn, err := pgx.Connect(context.Background(), "postgres://postgres:Roflan_2006@localhost:5432/data") // надо будет закинуть в gitignore и защитить от SQL инъекций, хз
	if err != nil{
		panic(err)
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(), "INSERT INTO data_user (username, password) VALUES ($1, $2)", username, password) // нужнот закинуть переменные, получаемые из строки в странице авторизации 
	if err != nil{
		log.Fatal(err)
	}

}
