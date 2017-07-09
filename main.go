package main

import (
	"log"
	"net/http"
	"text/template"
	"time"

	"./api"
)

// Decent reading: https://astaxie.gitbooks.io/build-web-application-with-golang/en/05.3.html

type HomePageData struct {
	Players []api.PlayerData
	Matches []api.MatchData
}

func renderHomePage(write http.ResponseWriter, req *http.Request) {
	tmplt, err := template.ParseFiles("static/home.html")
	if err != nil {
		write.WriteHeader(http.StatusInternalServerError)
		return
	}

	data := HomePageData{
		Players: api.GetAllPlayerData(),
		Matches: api.GetRecentMatches(),
	}
	err = tmplt.ExecuteTemplate(write, "home.html", data)
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		write.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type AddMatchPageData struct {
	Games   []api.GameData
	Players []api.PlayerData
}

func renderNewMatchPage(write http.ResponseWriter, req *http.Request) {
	tmplt, err := template.ParseFiles("static/addmatch.html")
	if err != nil {
		write.WriteHeader(http.StatusInternalServerError)
		return
	}

	data := AddMatchPageData{}
	data.Games = api.GetAllGameData()
	data.Players = api.GetAllPlayerData()
	err = tmplt.ExecuteTemplate(write, "addmatch.html", data)
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		write.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type AddPlayerPageData struct{}

func renderNewPlayerPage(write http.ResponseWriter, req *http.Request) {
	tmplt, err := template.ParseFiles("static/addplayer.html")
	if err != nil {
		write.WriteHeader(http.StatusInternalServerError)
		return
	}

	data := AddPlayerPageData{}
	err = tmplt.ExecuteTemplate(write, "addplayer.html", data)
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		write.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type AddGamePageData struct{}

func renderNewGamePage(write http.ResponseWriter, req *http.Request) {
	tmplt, err := template.ParseFiles("static/addgame.html")
	if err != nil {
		write.WriteHeader(http.StatusInternalServerError)
		return
	}

	data := AddGamePageData{}
	err = tmplt.ExecuteTemplate(write, "addgame.html", data)
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		write.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type HttpRequestHandler struct {
	mux map[string]func(http.ResponseWriter, *http.Request)
}

func (reqHandler *HttpRequestHandler) ServeHTTP(write http.ResponseWriter, req *http.Request) {
	log.Printf("[%s] src=%s dest=%s", req.Method, req.RemoteAddr, req.URL.Path)
	handlerFunc, ok := reqHandler.mux[req.URL.Path]
	if ok {
		handlerFunc(write, req)
		return
	}

	write.WriteHeader(http.StatusNotFound)
}

func main() {
	log.Print("Setting up web server...")
	handler := HttpRequestHandler{}
	handler.mux = make(map[string]func(http.ResponseWriter, *http.Request))
	handler.mux["/"] = renderHomePage
	handler.mux["/newgame"] = renderNewGamePage
	handler.mux["/newmatch"] = renderNewMatchPage
	handler.mux["/newplayer"] = renderNewPlayerPage

	handler.mux["/api/v1/get-rank-chart-data"] = api.GetRankHistory
	handler.mux["/api/v1/addgame"] = api.AddGame
	handler.mux["/api/v1/addmatch"] = api.AddMatch
	handler.mux["/api/v1/addplayer"] = api.AddPlayer

	server := http.Server{
		Addr:         ":8000",
		Handler:      &handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Print("Initializing API...")
	api.Initialize()
	log.Print("Running web server...")
	log.Fatal(server.ListenAndServe())
}
