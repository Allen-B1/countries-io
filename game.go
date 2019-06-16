package main

// Type Game represents a game
type Game struct {
//	Connections []*websocket.Connection // conn = [countryId] ???

	// The names of the countries
	Countries []string // name = [countryId]

	Terrain []int // countryId = [tileIndex]
	Armies []uint // army = [tileIndex]
	Cities map[int]bool // isCity = [tileIndex]
	Capitals map[int]bool // isCaptial = [tileIndex]

	// The current turn #
	Turn int
}

// Function NewGame creates and returns a new Game
func NewGame(countries []string, width int, height int) *Game {
	size := width * height
	return &Game{
		Countries: countries,
		Terrain: make([]int, size),
		Armies: make([]uint, size),
		Cities: make(map[int]bool),
		Capitals: make(map[int]bool),
		Turn: 0,
	}
}

// Method NextTurn
func (g *Game) NextTurn() {
	if g.Turn % 2 == 0 {
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
	g.Terrain[toTileIndex] = countryIndex
	g.Armies[toTileIndex] = g.Armies[fromTileIndex] - 1
	g.Armies[fromTileIndex] = 1
	return true
}
