package main

import (
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "style.css")
	})
	http.HandleFunc("/city.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "city.svg")
	})
	http.HandleFunc("/capital.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "capital.svg")
	})
	http.HandleFunc("/school.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "school.svg")
	})
	http.HandleFunc("/portal.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "portal.svg")
	})
	http.HandleFunc("/sound.wav", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "sound.wav")
	})

	http.HandleFunc("/ws/room", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		// wait for join command
		for {
			mt, msg, err := conn.ReadMessage()
			if _, ok := err.(*websocket.CloseError); ok {
				handleRoomCommand(conn, websocket.CloseMessage, nil)
				return
			}
			if err != nil {
				log.Println(err)
				return
			}
			args := strings.Fields(string(msg))
			handleRoomCommand(conn, mt, args)
		}
	})

	http.HandleFunc("/ws/game", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		// wait for join command
		for {
			mt, msg, err := conn.ReadMessage()
			if _, ok := err.(*websocket.CloseError); ok {
				handleGameCommand(conn, websocket.CloseMessage, nil)
				return
			}
			if err != nil {
				log.Println(err)
				return
			}
			args := strings.Fields(string(msg))
			handleGameCommand(conn, mt, args)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.Header().Set("Location", "/")
			w.WriteHeader(302)
			return
		}
		http.ServeFile(w, r, "index.html")
	})
	http.HandleFunc("/ffa", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "room.html")
	})
	http.HandleFunc("/1v1", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "room.html")
	})
	http.HandleFunc("/play", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "game.html")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.ListenAndServe(":"+port, nil)
}
