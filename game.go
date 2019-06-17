package main

import (
	"encoding/json"
	"math/rand"
	"sort"
)

const (
	// Tile constants for Game.Terrain
	TILE_EMPTY = -1
	TILE_WALL  = -2
)

// Type Game represents a game
type Game struct {
	// The names of the countries
	Countries []string // name = [countryId]

	Width  int
	Height int

	Terrain  []int        // countryId = [tileIndex]
	Armies   []uint       // army = [tileIndex]
	Cities   map[int]bool // isCity = [tileIndex]
	Capitals map[int]bool // isCaptial = [tileIndex]

	// The current turn #
	Turn int
}

// Function NewGame creates and returns a new Game
func NewGame(countries []string, width int, height int) *Game {
	size := width * height
	g := &Game{
		Countries: countries,
		Terrain:   make([]int, size),
		Armies:    make([]uint, size),
		Cities:    make(map[int]bool),
		Capitals:  make(map[int]bool),
		Turn:      0,
		Width:     width,
		Height:    height,
	}

	// Reset to -1
	for index, _ := range g.Terrain {
		g.Terrain[index] = TILE_EMPTY
	}

	for countryIndex, _ := range g.Countries {
		for {
			index := rand.Intn(size)
			if _, ok := g.Capitals[index]; ok {
				continue
			}
			g.Terrain[index] = countryIndex
			g.Capitals[index] = true
			break
			// TODO: 5x5 square around capital
		}
	}

	return g
}

// Method NextTurn
func (g *Game) NextTurn() {
	if g.Turn%2 == 0 {
		for cityIndex, _ := range g.Cities {
			g.Armies[cityIndex] += 1
		}
	}
	for capitalIndex, _ := range g.Capitals {
		g.Armies[capitalIndex] += 1
	}
	g.Turn++
}

// Method Attack causes a country to move armies
func (g *Game) Attack(countryIndex int, fromTileIndex int, toTileIndex int) bool {
	// TODO: Make legit
	// * Check if tiles are next to each other
	// * Compare army sizes
	// * Capturing cities/capitals = gaining 3x3/5x5 square
	// * Capture capitals => cities
	if g.Terrain[fromTileIndex] != countryIndex ||
		g.Armies[fromTileIndex] < 2 ||
		toTileIndex >= len(g.Terrain) || toTileIndex < 0 {
		return false
	}

	if g.Terrain[toTileIndex] == countryIndex {
		g.Armies[toTileIndex] += g.Armies[fromTileIndex] - 1
	} else {
		g.Armies[toTileIndex] = g.Armies[fromTileIndex] - 1
	}
	g.Armies[fromTileIndex] = 1
	g.Terrain[toTileIndex] = countryIndex
	return true
}

// Method MakeCity creates a city
func (g *Game) MakeCity(countryIndex int, tileIndex int) bool {
	if g.Terrain[tileIndex] != countryIndex || g.Armies[tileIndex] < 31 {
		return false
	}
	g.Armies[tileIndex] -= 30
	g.Cities[tileIndex] = true

	// TODO: +1 army in 3x3 square around city
	return true
}

func (g *Game) MakeWall(countryIndex int, tileIndex int) bool {
	if g.Terrain[tileIndex] != countryIndex {
		return false
	}
	g.Armies[tileIndex] = g.Armies[tileIndex] * 2
	g.Terrain[tileIndex] = TILE_WALL
	return true
}

// Method MarshalJSON implements the json.Marshaler interface
func (g *Game) MarshalJSON() ([]byte, error) {
	citylist := make([]int, 0, len(g.Cities))
	capitallist := make([]int, 0, len(g.Capitals))
	for city, _ := range g.Cities {
		citylist = append(citylist, city)
	}
	for capital, _ := range g.Capitals {
		capitallist = append(capitallist, capital)
	}
	sort.Ints(citylist)
	sort.Ints(capitallist)

	return json.Marshal(map[string]interface{}{
		"terrain":  g.Terrain,
		"armies":   g.Armies,
		"cities":   citylist,
		"capitals": capitallist,
	})
}
