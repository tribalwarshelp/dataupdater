package cron

import "github.com/tribalwarshelp/shared/models"

type tribeSearchableByID struct {
	*models.Tribe
}

func (t tribeSearchableByID) ID() int {
	return t.Tribe.ID
}

type playerSearchableByID struct {
	*models.Player
}

func (t playerSearchableByID) ID() int {
	return t.Player.ID
}

type searchableByID interface {
	ID() int
}

func makePlayersSearchable(players []*models.Player) []searchableByID {
	searchable := []searchableByID{}
	for _, player := range players {
		searchable = append(searchable, playerSearchableByID{player})
	}
	return searchable
}

func makeTribesSearchable(tribes []*models.Tribe) []searchableByID {
	searchable := []searchableByID{}
	for _, tribe := range tribes {
		searchable = append(searchable, tribeSearchableByID{tribe})
	}
	return searchable
}

func searchByID(haystack []searchableByID, id int) int {
	low := 0
	high := len(haystack) - 1

	for low <= high {
		median := (low + high) / 2

		if haystack[median].ID() < id {
			low = median + 1
		} else {
			high = median - 1
		}
	}

	if low == len(haystack) || haystack[low].ID() != id {
		return 0
	}

	return low
}
