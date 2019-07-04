/*
countries.io
Copyright (C) 2019 Allen B

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
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
	Join chan struct {
		int
		*websocket.Conn
	} // Incoming
	// Outgoing
	Error []chan string

	// Incoming
	Attack       [](chan [3]int)
	MakeCity     [](chan int)
	MakeWall     [](chan int)
	MakeSchool   [](chan int)
	MakePortal   [](chan int)
	Collect      [](chan int)
	MakeLauncher [](chan int)
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

		index, err := strconv.Atoi(args[2])
		if err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte("error "+err.Error()))
			return
		}

		thread, ok := gameThreads[gameId]
		if !ok {
			conn.WriteMessage(websocket.TextMessage, []byte("error game doesn't exist"))
		}
		thread.Join <- struct {
			int
			*websocket.Conn
		}{index, conn}
		select {
		case err := <-thread.Error[index]:
			conn.WriteMessage(websocket.TextMessage, []byte("error "+err))
		case <-time.After(500 * time.Millisecond):
		}
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
		if len(args) < 4 {
			return
		}
		fromTile, err1 := strconv.Atoi(args[1])
		toTile, err2 := strconv.Atoi(args[2])
		if err1 != nil || err2 != nil {
			log.Println("Error: ", err1, " or ", err2)
		}

		isHalf := 0
		if args[3] == "1" {
			isHalf = 1
		}

		select {
		case thread.Attack[info.Index] <- [3]int{fromTile, toTile, isHalf}:
		case <-time.After(300 * time.Millisecond):
		}
	case "city", "wall", "school", "portal", "collect", "launcher":
		if len(args) != 2 {
			return
		}
		tile, err := strconv.Atoi(args[1])
		if err != nil {
			log.Println(err)
		}
		channel := map[string](chan int){
			"city":     thread.MakeCity[info.Index],
			"wall":     thread.MakeWall[info.Index],
			"school":   thread.MakeSchool[info.Index],
			"portal":   thread.MakePortal[info.Index],
			"collect":  thread.Collect[info.Index],
			"launcher": thread.MakeLauncher[info.Index],
		}[args[0]]

		channel <- tile
	}
}

func startGameThread(gameId string, game *Game) {
	thread := gameThread{}
	thread.Join = make(chan struct {
		int
		*websocket.Conn
	})
	for _, _ = range game.Countries {
		thread.Error = append(thread.Error, make(chan string))
		thread.Attack = append(thread.Attack, make(chan [3]int))
		thread.MakeCity = append(thread.MakeCity, make(chan int, 16))
		thread.MakeWall = append(thread.MakeWall, make(chan int, 16))
		thread.MakeSchool = append(thread.MakeSchool, make(chan int, 16))
		thread.MakePortal = append(thread.MakePortal, make(chan int, 16))
		thread.Collect = append(thread.Collect, make(chan int, 16))
		thread.MakeLauncher = append(thread.MakeLauncher, make(chan int, 16))
	}

	gameThreads[gameId] = thread

	n := 1
	// wait for all to join
	for n <= len(game.Countries) {
		data := <-thread.Join
		index := data.int
		conn := data.Conn

		log.Printf("%d joined\n", index)

		if index < 0 {
			continue
		}

		// Check if somebody already connected as that player
		gameConns.Lock()
		for _, info := range gameConns.Map {
			if info.Game == gameId && info.Index == index {
				thread.Error[index] <- "somebody took your place"
				gameConns.Unlock()
				continue
			}
		}

		gameConns.Map[conn] = gameConnInfo{Game: gameId, Index: index}
		gameConns.Unlock()
		n++
	}

	broadcastGame(gameId, "player_list "+strings.Join(game.Countries, " "))
	broadcastGame(gameId, fmt.Sprintf("map %d %d", game.Width, game.Height))
	log.Println("Broadcasted " + gameId)

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	oldterrain := make([]int, 0)
	oldarmies := make([]uint, 0)

	turn := true
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
			// go through gameConnInfos
			return
		}

		<-ticker.C
		if turn {
			game.NextTurn()
		}
		turn = !turn

		// Read attacks
		for countryIndex, attack := range thread.Attack {
			select {
			case data := <-attack:
				game.Attack(countryIndex, data[0], data[1], data[2] != 0)
			default:
			}
		}

		for countryIndex, channel := range thread.MakeWall {
		loopwall:
			for {
				select {
				case data := <-channel:
					game.MakeWall(countryIndex, data)
				default:
					break loopwall
				}
			}
		}

		for countryIndex, channel := range thread.MakeCity {
		loopcity:
			for {
				select {
				case data := <-channel:
					game.MakeCity(countryIndex, data)
				default:
					break loopcity
				}
			}
		}

		for countryIndex, channel := range thread.MakeSchool {
		loopschool:
			for {
				select {
				case data := <-channel:
					game.MakeSchool(countryIndex, data)
				default:
					break loopschool
				}
			}
		}

		for countryIndex, channel := range thread.MakePortal {
		loopportal:
			for {
				select {
				case data := <-channel:
					game.MakePortal(countryIndex, data)
				default:
					break loopportal
				}
			}
		}

		for countryIndex, channel := range thread.Collect {
		loopcollect:
			for {
				select {
				case data := <-channel:
					game.Collect(countryIndex, data)
				default:
					break loopcollect
				}
			}
		}

		for countryIndex, channel := range thread.MakeLauncher {
		looplauncher:
			for {
				select {
				case data := <-channel:
					game.MakeLauncher(countryIndex, data)
				default:
					break looplauncher
				}
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
