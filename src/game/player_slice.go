// Generated by: gen
// TypeWriter: slice
// Directive: +gen on *Player

package game

import "errors"

// PlayerSlice is a slice of type *Player. Use it where you would use []*Player.
type PlayerSlice []*Player

// First returns the first element that returns true for the passed func. Returns error if no elements return true. See: http://clipperhouse.github.io/gen/#First
func (rcv PlayerSlice) First(fn func(*Player) bool) (result *Player, err error) {
	for _, v := range rcv {
		if fn(v) {
			result = v
			return
		}
	}
	err = errors.New("no PlayerSlice elements return true for passed func")
	return
}
