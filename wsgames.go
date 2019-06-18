package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var games = make(map[string]*Game)

type gameConnInfo struct {
	Game  string
	Index int
}

var gameConns = make(map[*websocket.Conn]gameConnInfo)

func broadcastGame(gameId string, message string) {
	for conn, info := range gameConns {
		if info.Game == gameId {
			conn.WriteMessage(websocket.TextMessage, []byte(message))
		}
	}
}

type gameThread struct {
	Attack [](chan [2]int)
}

var gameThreads = make(map[string]gameThread)

func handleGameCommand(conn *websocket.Conn, mt int, args []string) {
	if mt != websocket.CloseMessage && len(args) == 0 {
		return
	}
	if mt == websocket.TextMessage && args[0] == "join" {
		_, ok := gameConns[conn]
		if ok {
			return
		}
		if len(args) < 3 {
			return
		}
		gameId := args[1]
		// Check if game exists
		if _, ok := games[gameId]; !ok {
			conn.WriteMessage(websocket.TextMessage, []byte("error game doesn't exist"))
			return
		}

		index, err := strconv.Atoi(args[2])
		if err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte("error "+err.Error()))
			return
		}

		if index >= 0 {
			for _, info := range gameConns {
				if info.Game == gameId && info.Index == index {
					conn.WriteMessage(websocket.TextMessage, []byte("error somebody took your place"))
					return
				}
			}
		}

		gameConns[conn] = gameConnInfo{Game: gameId, Index: index}
		game := games[gameId]

		conn.WriteMessage(websocket.TextMessage, []byte("player_list "+strings.Join(game.Countries, " ")))
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("map %d %d", game.Width, game.Height)))
		return
	}

	info, ok := gameConns[conn]
	if !ok {
		return
	}

	if info.Index < 0 {
		// actions are not allowed
		return
	}

	game, ok := games[info.Game]
	if !ok {
		return
	}
	thread := gameThreads[info.Game]

	if mt == websocket.CloseMessage {
		game.Leave(info.Index)
		return
	}

	switch args[0] {
	case "attack":
		if len(args) != 3 {
			return
		}
		fromTile, err1 := strconv.Atoi(args[1])
		toTile, err2 := strconv.Atoi(args[2])
		if err1 != nil || err2 != nil {
			log.Println("Error: ", err1, " or ", err2)
		}
		select {
		case thread.Attack[info.Index] <- [2]int{fromTile, toTile}:
		case <-time.After(500 * time.Millisecond):
		}
	case "city", "wall":
		if len(args) != 2 {
			return
		}
		tile, err := strconv.Atoi(args[1])
		if err != nil {
			log.Println(err)
		}
		if args[0] == "city" {
			game.MakeCity(info.Index, tile)
		} else if args[0] == "wall" {
			game.MakeWall(info.Index, tile)
		}
	}
}

func startGameThread(gameId string, game *Game) {
	thread := gameThread{}
	for _, _ = range game.Countries {
		thread.Attack = append(thread.Attack, make(chan [2]int))
	}

	gameThreads[gameId] = thread

	dur := 500 * time.Millisecond
	old := (*Game)(nil)
	for {
		// broadcast update
		data, err := game.MarshalJSON(old)
		if err != nil {
			log.Println(err)
			continue
		}
		broadcastGame(gameId, "update "+string(data))
		old = game
		time.Sleep(dur)
		game.NextTurn()

		// Read attacks
		for countryIndex, attack := range thread.Attack {
			select {
			case data := <-attack:
				game.Attack(countryIndex, data[0], data[1])
			default:
			}
		}

		loserstr := ""
		for loser, _ := range game.Losers {
			loserstr += " " + fmt.Sprint(loser)
		}

		broadcastGame(gameId, "player_lose"+(loserstr))

		// if only one person left stop
		if len(game.Countries)-len(game.Losers) <= 1 {
			delete(games, gameId)
			delete(gameThreads, gameId)
			return
		}
	}
}
