package main

// Type Room represents a room
type Room struct {
	Max int // Max # of people

	Countries map[string]bool
}

func NewRoom(max int) *Room {
	r := &Room{
		Max: max,
		Countries: make(map[string]bool),
	}

	return r
}

// Add a player
func (r *Room) Add(name string) bool {
	if len(r.Countries) >= r.Max {
		return false
	}
	if r.Countries[name] { // Already somebody here
		return false
	}
	r.Countries[name] = true
	return true
}

// Remove a player
func (r *Room) Remove(name string) bool {
	delete(r.Countries, name)
	return true
}

func (r *Room) Game() *Game {
	countrylist := make([]string, 0, len(r.Countries))
	for country, _ := range r.Countries {
		countrylist = append(countrylist, country)
	}
	game := NewGame(countrylist, 20, 20)
	return game
}

