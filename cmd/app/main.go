package main

import (
	"log"

	"github.com/joho/godotenv"
	"Messanger/cmd/handlers"
	"Messanger/internal/database"
)

func main(){
	err := godotenv.Load()
	if err != nil{
		log.Println("Can't connect to .env file!")
	}
	database.InitDB() // Проверка на базу данных
	handlers.HandleRequests() 
}