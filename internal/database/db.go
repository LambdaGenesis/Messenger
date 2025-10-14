package database

import (
	"os"
	"time"
	"log"
	"net/http"
	"context"
	"strconv"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/labstack/echo/v4"
)

func RegPage(c echo.Context) error { 	
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
	WriteSQL(getUsernameReg, getPasswordReg)
	return c.Render(http.StatusOK, "reg_page", nil)
}
func AuthPage(c echo.Context) error{ 
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
func InitDB(){
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
		InitDB() // Рекурсия на проверку 
		return
	}
}
// Запись информации о клиенте в базу данных
func WriteSQL(username, password string) {
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