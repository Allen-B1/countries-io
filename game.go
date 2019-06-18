package main

import (
	"encoding/json"
	"log"
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

	Losers map[int]bool // People who lost

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
		Losers:    make(map[int]bool),
		Turn:      0,
		Width:     width,
		Height:    height,
	}

	// Reset to -1
	for index, _ := range g.Terrain {
		g.Terrain[index] = TILE_EMPTY
	}

	for countryIndex, _ := range g.Countries {
	makecapital:
		for {
			index := rand.Intn(size)
			if _, ok := g.Capitals[index]; ok {
				continue
			}
			for _, tileAround := range g.TilesAround(index, 18) {
				if g.Capitals[tileAround] {
					log.Println("Capital too close, getting another")
					continue makecapital
				}
			}

			g.Terrain[index] = countryIndex
			g.Capitals[index] = true
			g.ConvertAround(index, 2, countryIndex, TILE_EMPTY)

			break
		}
	}

	return g
}

// Method NextTurn
func (g *Game) NextTurn() {
outer:
	for index, terrain := range g.Terrain {
		// Don't increase anything for not-in-game-anymore people
		for loser, _ := range g.Losers {
			if terrain == loser {
				continue outer
			}
		}

		switch g.TileType(index) {
		case TILE_WALL:
			continue
		case TILE_EMPTY:
			continue
		case TILE_RURAL:
			if g.Turn%50 == 0 && g.Turn != 0 {
				g.Armies[index] += 1
			}
			continue
		case TILE_SUBURB:
			if g.Turn%20 == 0 && g.Turn != 0 {
				g.Armies[index] += 1
			}
		case TILE_URBAN:
			if g.Turn%2 == 0 && g.Cities[index] {
				g.Armies[index] += 1
			}
			if g.Capitals[index] {
				g.Armies[index] += 1
			}
		}
	}
	g.Turn++
}

// Method Attack causes a country to move armies
func (g *Game) Attack(countryIndex int, fromTileIndex int, toTileIndex int) bool {
	// TODO: Make legit
	// * Check if tiles are next to each other
	// * Capturing cities/capitals = gaining 3x3/5x5 square
	if g.Terrain[fromTileIndex] != countryIndex ||
		g.Armies[fromTileIndex] < 2 ||
		toTileIndex >= len(g.Terrain) || toTileIndex < 0 {
		return false
	}

	fromRow := fromTileIndex / g.Width
	toRow := toTileIndex / g.Width
	fromCol := fromTileIndex % g.Width
	toCol := toTileIndex % g.Width
	if fromRow != toRow && fromCol != toCol { // TODO: Expand
		return false
	}

	if g.Terrain[toTileIndex] == countryIndex {
		g.Armies[toTileIndex] += g.Armies[fromTileIndex] - 1
		g.Terrain[toTileIndex] = countryIndex
	} else {
		toCountry := g.Terrain[toTileIndex]
		if g.Armies[fromTileIndex]-1 > g.Armies[toTileIndex] { // win
			g.Armies[toTileIndex] = g.Armies[fromTileIndex] - 1 - g.Armies[toTileIndex]

			if g.Cities[toTileIndex] {
				g.ConvertAround(toTileIndex, 1, countryIndex, g.Terrain[toTileIndex])
			}

			if g.Capitals[toTileIndex] {
				g.ConvertAround(toTileIndex, 2, countryIndex, g.Terrain[toTileIndex])

				delete(g.Capitals, toTileIndex)
				g.Cities[toTileIndex] = true
			}

			g.Terrain[toTileIndex] = countryIndex
		} else if g.Armies[fromTileIndex]-1 < g.Armies[toTileIndex] { // lose
			g.Armies[toTileIndex] -= g.Armies[fromTileIndex] - 1
		} else if g.Armies[fromTileIndex]-1 == g.Armies[toTileIndex] { // tie
			g.Armies[toTileIndex] = 0
			g.Terrain[toTileIndex] = TILE_EMPTY
		}

		if toCountry >= 0 {
			g.checkLoss(toCountry)
		}
	}

	g.Armies[fromTileIndex] = 1
	return true
}

// Method MakeCity creates a city
func (g *Game) MakeCity(countryIndex int, tileIndex int) bool {
	if g.Terrain[tileIndex] != countryIndex || g.Armies[tileIndex] < 31 || g.Cities[tileIndex] {
		return false
	}
	for _, tile := range g.TilesAround(tileIndex, 4) {
		if g.Cities[tile] || g.Capitals[tile] {
			return false // Can't make a city too close to a city/capital
		}
	}

	g.Armies[tileIndex] -= 30
	g.Cities[tileIndex] = true
	g.ConvertAround(tileIndex, 1, countryIndex, TILE_EMPTY)
	return true
}

func (g *Game) MakeWall(countryIndex int, tileIndex int) bool {
	if g.Terrain[tileIndex] != countryIndex {
		return false
	}
	g.Armies[tileIndex] *= 5
	g.Terrain[tileIndex] = TILE_WALL
	return true
}

func (g *Game) Leave(countryIndex int) {
	g.Losers[countryIndex] = true
}

// TODO: Make this actually do stuff
func createDiff(old []int, new_ []int) []int {
	out := []int{0, len(old)}
	out = append(out, new_...)
	return out
}

// Method MarshalJSON creates json
func (g *Game) MarshalJSON(old *Game) ([]byte, error) {
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

	if old == nil {
		old = &Game{
			Terrain: nil,
			Armies:  nil,
		}
	}
	terraindiff := createDiff(old.Terrain, g.Terrain)
	armiesold := make([]int, 0)
	for _, army := range old.Armies {
		armiesold = append(armiesold, int(army))
	}
	armiesnew := make([]int, 0)
	for _, army := range g.Armies {
		armiesnew = append(armiesnew, int(army))
	}
	armiesdiff := createDiff(armiesold, armiesnew)

	return json.Marshal(map[string]interface{}{
		"terrain_diff": terraindiff,
		"armies_diff":  armiesdiff,
		"cities":       citylist,
		"capitals":     capitallist,
	})
}

func (g *Game) TilesAround(tile int, r int) []int {
	out := make([]int, 0)
	tileCol := tile % g.Width
	tileRow := tile / g.Width
	startCol := tileCol - r
	endCol := tileCol + r
	startRow := tileRow - r
	endRow := tileRow + r

	if startCol < 0 {
		startCol = 0
	}
	if endCol > g.Width-1 {
		endCol = g.Width - 1
	}
	if startRow < 0 {
		startRow = 0
	}
	if endRow > g.Height-1 {
		endRow = g.Height - 1
	}

	for row := startRow; row <= endRow; row++ {
		for col := startCol; col <= endCol; col++ {
			out = append(out, row*g.Width+col)
		}
	}

	return out
}

func (g *Game) ConvertAround(tile int, r int, countryIndex int, fromCountryIndex int) {
	for _, tileAround := range g.TilesAround(tile, r) {
		if g.Terrain[tileAround] != fromCountryIndex {
			continue
		}
		g.Terrain[tileAround] = countryIndex
		if g.Armies[tileAround] == 0 {
			g.Armies[tileAround] = 1
		}
	}
}

const (
	TILE_RURAL  = 0
	TILE_SUBURB = 1
	TILE_URBAN  = 2
)

func (g *Game) TileType(tile int) int {
	if g.Terrain[tile] < 0 {
		return g.Terrain[tile]
	}
	if g.Capitals[tile] || g.Cities[tile] {
		return TILE_URBAN
	}
	for capital, _ := range g.Capitals {
		if g.Terrain[capital] == g.Terrain[tile] {
			for _, tileAround := range g.TilesAround(capital, 2) {
				if tileAround == tile {
					return TILE_SUBURB
				}
			}
		}
	}
	for city, _ := range g.Cities {
		if g.Terrain[city] == g.Terrain[tile] {
			for _, tileAround := range g.TilesAround(city, 1) {
				if tileAround == tile {
					return TILE_SUBURB
				}
			}
		}
	}
	return TILE_RURAL
}

func (g *Game) checkLoss(countryIndex int) {
	log.Println(countryIndex, "lost?")
	if g.Losers[countryIndex] {
		return
	}
	for index, terrain := range g.Terrain {
		if terrain == countryIndex {
			log.Println("nope,", index)
			return // not lost yet
		}
	}
	log.Println(countryIndex, "lost!")
	g.Losers[countryIndex] = true
}
