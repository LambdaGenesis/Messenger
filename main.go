package main

import (
	"context"
	_ "fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"

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
	
	e.GET("/", mainPage)
	e.GET("/home", homePage)
	e.GET("/auth", showAuthPage)
	e.POST("/auth/post", authPage)
	e.GET("/about", aboutPage)
	e.POST("/reg/post", regPage)
	e.GET("/reg", showRegPage)
	e.GET("/contacts", contactsPage)
	
	e.Logger.Fatal(e.Start(":8080"))
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
		"Title": "Contacts",
	})
}

func showRegPage(c echo.Context) error{
	return c.Render(http.StatusOK, "reg_page", map[string]interface{}{
        "Title": "Registration",
        "Error": "", // added empty error for a template
    })
}

func regPage(c echo.Context) error {
	if c.Request().Method != http.MethodPost{
		return c.Redirect(http.StatusFound, "/reg")
	}
	
	getUsernameReg := c.FormValue("usernameReg")
	getPasswordReg := c.FormValue("passwordReg")

	writeSQL(getUsernameReg, getPasswordReg)

	data := struct{Error string}{Error: "Password or login is already exists"}
	return c.Render(http.StatusOK, "reg_page", data)
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

	// TEST
	conn, err := pgx.Connect(context.Background(), "postgres://postgres:Roflan_2006@localhost:5432/data")
	if err != nil{
		panic(err)
	}
	defer conn.Close(context.Background())
	
	rows, err := conn.Query(context.Background(), "SELECT username, password FROM data_user")
	if err != nil{
		log.Fatal(err)
	}
	defer rows.Close()

	var username string
	var password int
	
	for rows.Next(){
		err := rows.Scan(&username, &password)
		if err != nil{
			log.Fatal(err)
		}
		stringPassword := strconv.Itoa(password)
		if getUsernameAuth == username && getPasswordAuth == stringPassword{
			return c.Redirect(http.StatusFound, "/home")
		}
	}

	data := struct{ Error string }{Error: "Wrong password or login"}
	return c.Render(http.StatusOK, "auth_page", data)
}

func writeSQL(username, password string){
	conn, err := pgx.Connect(context.Background(), "postgres://postgres:Roflan_2006@localhost:5432/data") // надо будет закинуть в gitignore и защитить от SQL инъекций, хз
	if err != nil{
		log.Fatal(err)
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(), "INSERT INTO data_user (username, password) VALUES ($1, $2)", username, password) // нужно закинуть переменные, получаемые из строки в странице авторизации 
	if err != nil{
		log.Fatal(err)
	}

}

