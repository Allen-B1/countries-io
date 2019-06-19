package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

var rooms = make(map[string]*Room)

// Returns the room. If not found creates one
func roomsGet(id string) *Room {
	room, ok := rooms[id]
	if !ok {
		room = NewRoom(6)
		rooms[id] = room
		go roomThread(id, room)
	}

	return room
}

type roomConnInfo struct {
	Room    string
	Country string
}

var roomConns = struct {
	Map map[*websocket.Conn]roomConnInfo
	sync.Mutex
}{
	Map: make(map[*websocket.Conn]roomConnInfo),
}

func broadcastRoom(roomId string, message string) {
	for conn, info := range roomConns.Map {
		if roomId == info.Room {
			conn.WriteMessage(websocket.TextMessage, []byte(message))
		}
	}
}

func handleRoomCommand(conn *websocket.Conn, mt int, args []string) {
	roomConns.Lock()
	defer roomConns.Unlock()
	if mt == websocket.CloseMessage {
		info, ok := roomConns.Map[conn]
		if !ok {
			return
		}
		roomId := info.Room
		country := info.Country
		room := rooms[roomId]
		if room != nil {
			room.Remove(country)

			delete(roomConns.Map, conn)
			broadcastRoom(roomId, "player_remove")

			log.Println("leave " + roomId + " " + country)
			if room.StartTime == nil {
				broadcastRoom(roomId, "time_reset")
			}
		}
		return
	}
	if mt == websocket.TextMessage && len(args) >= 1 && args[0] == "ping" {
		conn.WriteMessage(websocket.TextMessage, []byte("pong"))
	}
	if mt == websocket.TextMessage && len(args) >= 3 && args[0] == "join" {
		if _, ok := roomConns.Map[conn]; ok {
			conn.WriteMessage(websocket.TextMessage, []byte("error join error: already in a game"))
			return
		}

		roomId := args[1]
		room := roomsGet(roomId)
		if !room.Add(args[2]) {
			conn.WriteMessage(websocket.TextMessage, []byte("error join error: that country already exists"))
			return
		}

		roomConns.Map[conn] = roomConnInfo{
			Room:    args[1],
			Country: args[2],
		}
		conn.WriteMessage(websocket.TextMessage, []byte("player_max "+fmt.Sprint(room.Max)))
		if len(room.Countries)-1 > 0 {
			conn.WriteMessage(websocket.TextMessage, []byte("player_add "+fmt.Sprint(len(room.Countries)-1)))
		}
		broadcastRoom(args[1], "player_add 1")
		if room.StartTime != nil {
			broadcastRoom(args[1], "time "+fmt.Sprint(room.StartTime.Unix()*1000))
		} else {
			broadcastRoom(args[1], "time_reset")
		}

		if len(room.Countries) == room.Max {
			startGame(roomId, room)
		}

		log.Println("join " + args[1] + " " + args[2])
		return
	}
}

func roomThread(roomId string, room *Room) {
	for {
		time.Sleep(1 * time.Second)
		if room.StartTime != nil && time.Now().After(*room.StartTime) {
			log.Println("Starting...")
			startGame(roomId, room)
		}
	}
}

func startGame(roomId string, room *Room) {
	game := room.Game()
	// broadcast start
	gameId := strconv.FormatInt(rand.Int63(), 36)
	games[gameId] = game

	for conn, info := range roomConns.Map {
		if roomId == info.Room {
			index := -1
			for i, country := range game.Countries {
				if country == info.Country {
					index = i
				}
			}
			conn.WriteMessage(websocket.TextMessage, []byte("start "+gameId+" "+fmt.Sprint(index)))
		}
	}

	go startGameThread(gameId, game)
}
