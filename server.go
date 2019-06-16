package main

import (
	"net/http"
	"github.com/gorilla/websocket"
	"log"
	"fmt"
	"strings"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var rooms = make(map[string]*Room)

// Returns the room. If not found creates one
func roomsGet(id string) *Room {
	room, ok := rooms[id]
	if !ok {
		room = NewRoom(4)
		rooms[id] = room
	}

	return room
}

type connInfo struct {
	Room string
	Country string
}

var roomConns = make(map[*websocket.Conn]connInfo)

func broadcastRoom(roomId string, message string) {
	for conn, info := range roomConns {
		if roomId == info.Room {
			conn.WriteMessage(websocket.TextMessage, []byte(message))
		}
	}
}

func handleRoomCommand(conn *websocket.Conn, mt int, args []string) {
	if mt == websocket.CloseMessage {
		info, ok := roomConns[conn]
		if !ok {
			return
		}
		roomId := info.Room
		country := info.Country
		room := rooms[roomId]
		if room != nil {
			room.Remove(country)
			delete(roomConns, conn)
			broadcastRoom(roomId, "player_remove")

			log.Println("leave " + roomId + " " + country)
		}
		return
	}
	if mt == websocket.TextMessage && args[0] == "join" {
		if _, ok := roomConns[conn]; ok {
			conn.WriteMessage(websocket.TextMessage, []byte("error join error: already in a game"))
			return
		}

		room := roomsGet(args[1])
		if !room.Add(args[2]) {
			conn.WriteMessage(websocket.TextMessage, []byte("error join error: that country already exists"))
			return
		}
		if len(room.Countries) == room.Max {
			game := room.Game()
			// broadcast start
			_ = game
		}
		roomConns[conn] = connInfo {
			Room: args[1],
			Country: args[2],
		}
		if len(room.Countries) - 1 > 0 {
			conn.WriteMessage(websocket.TextMessage, []byte("player_add " + fmt.Sprint(len(room.Countries) - 1)))
		}
		broadcastRoom(args[1], "player_add 1")

		log.Println("join " + args[1] + " " + args[2])
		return
 	}
}

func main() {
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

	http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	http.HandleFunc("/ffa", func (w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "ffa.html")
	})
	http.ListenAndServe(":8080", nil)
}

