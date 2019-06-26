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
	"log"
	"time"
)

// Type Room represents a room
type Room struct {
	Max int // Max # of people

	Countries map[string]bool

	StartTime *time.Time
}

func NewRoom(max int) *Room {
	r := &Room{
		Max:       max,
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
	if len(r.Countries) >= 2 && r.StartTime == nil {
		r.StartTime = new(time.Time)
		*r.StartTime = time.Now().Add(time.Duration(2 * time.Minute))
		log.Println(*r.StartTime)
	}
	return true
}

// Remove a player
func (r *Room) Remove(name string) bool {
	delete(r.Countries, name)
	if len(r.Countries) <= 1 {
		r.StartTime = nil
	}
	return true
}

func (r *Room) Game() *Game {
	countrylist := make([]string, 0, len(r.Countries))
	for country, _ := range r.Countries {
		countrylist = append(countrylist, country)
	}
	return NewGame(countrylist, (len(countrylist)+4)*10, (len(countrylist)+4)*10)
}
