package main

import (
	"html/template"
	"net/http"
)

func homePage(w http.ResponseWriter, r *http.Request){
	tmpl, err := template.ParseFiles("templates/home_page.html", "templates/header.html", "templates/footer.html")
	if err != nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "home_page", nil)
	if err != nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func mainPage(w http.ResponseWriter, r *http.Request){
	tmpl, err := template.ParseFiles("templates/main_page.html", "templates/header.html", "templates/footer.html")
	if err != nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "main_page", nil)
	if err != nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func showAuthPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles("templates/auth_page.html", "templates/footer.html", "templates/header.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "auth_page", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func authPage(w http.ResponseWriter, r *http.Request){
	if r.Method != http.MethodPost {
        http.Redirect(w, r, "/auth", http.StatusFound)
        return
    }

	username := r.FormValue("username")
	password := r.FormValue("password")

	tmpl, err := template.ParseFiles("templates/auth_page.html", "templates/footer.html", "templates/header.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	if username == "aaa" && password == "aaa"{
		http.Redirect(w, r, "/home", http.StatusFound)
		return
	}
	data := struct{ Error string }{Error: "Неверный логин или пароль"}

	err = tmpl.ExecuteTemplate(w, "auth_page", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func HandleRequests(){
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	
	http.HandleFunc("/", mainPage)
	http.HandleFunc("/auth", showAuthPage)
	http.HandleFunc("/auth/post", authPage)
	http.HandleFunc("/home", homePage)
	http.ListenAndServe(":8080", nil)
}

func main(){
	HandleRequests()
}