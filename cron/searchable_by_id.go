package cron

import "github.com/tribalwarshelp/shared/models"

type tribeSearchableByID struct {
	*models.Tribe
}

func (t tribeSearchableByID) id() int {
	return t.ID
}

type playerSearchableByID struct {
	*models.Player
}

func (t playerSearchableByID) id() int {
	return t.ID
}

type searchableByID interface {
	id() int
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

		if haystack[median].id() < id {
			low = median + 1
		} else {
			high = median - 1
		}
	}

	if low == len(haystack) || haystack[low].id() != id {
		return 0
	}

	return low
}
