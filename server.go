package main

import (
	"net/http"
	"github.com/gorilla/websocket"
	"log"
	"strconv"
	"strings"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var games = make(map[string]*Game)

type gameConnInfo struct {
	Game string
	Index int
}

var gameConns = make(map[*websocket.Conn]gameConnInfo)

func handleGameCommand(conn *websocket.Conn, mt int, args []string) {
	if mt == websocket.CloseMessage {

	}
	if len(args) == 0 {
		return
	}
	switch args[0] {
	case "join":
		_, ok := gameConns[conn]
		if ok {
			return
		}
		if len(args) < 3 {
			return
		}
		gameId := args[0]
		index, err := strconv.Atoi(args[1])
		if err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte("error " + err.Error()))
		}
		gameConns[conn] = gameConnInfo{Game: gameId, Index: index}
		conn.WriteMessage(websocket.TextMessage, []byte("start"))
	}
}

func main() {
	http.HandleFunc("/style.css", func (w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "style.css")
	})

	http.HandleFunc("/ws/room", func (w http.ResponseWriter, r *http.Request) {
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

	http.HandleFunc("/ws/game", func (w http.ResponseWriter, r *http.Request) {
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

	http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	http.HandleFunc("/ffa", func (w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ffa" {
			http.ServeFile(w, r, "room.html")
		} else {
			w.WriteHeader(404)
		}
	})
	http.HandleFunc("/play", func (w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "game.html")
	})
	http.ListenAndServe(":8080", nil)
}

