package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var games = make(map[string]*Game)

type gameConnInfo struct {
	Game  string
	Index int
}

var gameConns = struct {
	Map map[*websocket.Conn]gameConnInfo
	sync.Mutex
}{
	Map: make(map[*websocket.Conn]gameConnInfo),
}

func broadcastGame(gameId string, message string) {
	gameConns.Lock()
	for conn, info := range gameConns.Map {
		if info.Game == gameId {
			conn.WriteMessage(websocket.TextMessage, []byte(message))
		}
	}
	gameConns.Unlock()
}

type gameThread struct {
	Attack     [](chan [2]int)
	MakeCity   [](chan int)
	MakeWall   [](chan int)
	MakeSchool [](chan int)
}

var gameThreads = make(map[string]gameThread)

func handleGameCommand(conn *websocket.Conn, mt int, args []string) {
	if mt != websocket.CloseMessage && len(args) == 0 {
		return
	}
	if mt == websocket.TextMessage && args[0] == "join" {
		gameConns.Lock()
		_, ok := gameConns.Map[conn]
		gameConns.Unlock()
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
			gameConns.Lock()
			for _, info := range gameConns.Map {
				if info.Game == gameId && info.Index == index {
					conn.WriteMessage(websocket.TextMessage, []byte("error somebody took your place"))
					gameConns.Unlock()
					return
				}
			}
			gameConns.Unlock()
		}

		gameConns.Lock()
		gameConns.Map[conn] = gameConnInfo{Game: gameId, Index: index}
		gameConns.Unlock()

		game := games[gameId]

		conn.WriteMessage(websocket.TextMessage, []byte("player_list "+strings.Join(game.Countries, " ")))
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("map %d %d", game.Width, game.Height)))
		return
	}

	gameConns.Lock()
	info, ok := gameConns.Map[conn]
	gameConns.Unlock()
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
	case "city", "wall", "school":
		if len(args) != 2 {
			return
		}
		tile, err := strconv.Atoi(args[1])
		if err != nil {
			log.Println(err)
		}
		if args[0] == "city" {
			select {
			case thread.MakeCity[info.Index] <- tile:
			case <-time.After(500 * time.Millisecond):
			}
		} else if args[0] == "wall" {
			select {
			case thread.MakeWall[info.Index] <- tile:
			case <-time.After(500 * time.Millisecond):
			}
		} else if args[0] == "school" {
			select {
			case thread.MakeSchool[info.Index] <- tile:
			case <-time.After(500 * time.Millisecond):
			}
		}
	}
}

func startGameThread(gameId string, game *Game) {
	thread := gameThread{}
	for _, _ = range game.Countries {
		thread.Attack = append(thread.Attack, make(chan [2]int))
		thread.MakeCity = append(thread.MakeCity, make(chan int))
		thread.MakeWall = append(thread.MakeWall, make(chan int))
		thread.MakeSchool = append(thread.MakeSchool, make(chan int))
	}

	gameThreads[gameId] = thread

	// wait for all to join
	for {
		n := 0
		gameConns.Lock()
		for _, info := range gameConns.Map {
			if info.Game == gameId {
				n++
			}
		}
		gameConns.Unlock()
		if n >= len(game.Countries) {
			break
		}
	}

	dur := 250 * time.Millisecond
	oldterrain := make([]int, 0)
	oldarmies := make([]uint, 0)
	turn := false
	for {
		// broadcast update
		data, err := game.MarshalJSON(oldterrain, oldarmies)
		if err != nil {
			log.Println(err)
			continue
		}
		broadcastGame(gameId, "update "+string(data))
		if len(oldterrain) != len(game.Terrain) {
			oldterrain = append([]int(nil), game.Terrain...)
		} else {
			copy(oldterrain, game.Terrain)
		}
		if len(oldarmies) != len(game.Armies) {
			oldarmies = append([]uint(nil), game.Armies...)
		} else {
			copy(oldarmies, game.Armies)
		}

		// if only one person left stop
		if len(game.Countries)-len(game.Losers) <= 1 {
			delete(games, gameId)
			delete(gameThreads, gameId)
			return
		}

		time.Sleep(dur)
		turn = !turn
		if turn {
			game.NextTurn()
		}

		// Read attacks
		for countryIndex, attack := range thread.Attack {
			select {
			case data := <-attack:
				game.Attack(countryIndex, data[0], data[1])
			default:
			}
		}

		for countryIndex, channel := range thread.MakeWall {
			select {
			case data := <-channel:
				game.MakeWall(countryIndex, data)
			default:
			}
		}

		for countryIndex, channel := range thread.MakeCity {
			select {
			case data := <-channel:
				game.MakeCity(countryIndex, data)
			default:
			}
		}

		for countryIndex, channel := range thread.MakeSchool {
			select {
			case data := <-channel:
				game.MakeSchool(countryIndex, data)
			default:
			}
		}

		if len(game.Losers) != 0 {
			loserstr := ""
			for loser, _ := range game.Losers {
				loserstr += " " + fmt.Sprint(loser)
			}

			broadcastGame(gameId, "player_lose"+(loserstr))
		}
	}
}
