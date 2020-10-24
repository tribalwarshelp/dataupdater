package cron

import "github.com/tribalwarshelp/shared/models"

type tribesSearchableByID struct {
	tribes []*models.Tribe
}

func (searchable tribesSearchableByID) GetID(index int) int {
	return searchable.tribes[index].ID
}

func (searchable tribesSearchableByID) Len() int {
	return len(searchable.tribes)
}

type playersSearchableByID struct {
	players []*models.Player
}

func (searchable playersSearchableByID) GetID(index int) int {
	return searchable.players[index].ID
}

func (searchable playersSearchableByID) Len() int {
	return len(searchable.players)
}

type searchableByID interface {
	GetID(index int) int
	Len() int
}

func makePlayersSearchable(players []*models.Player) searchableByID {
	return playersSearchableByID{
		players: players,
	}
}

func makeTribesSearchable(tribes []*models.Tribe) searchableByID {
	return tribesSearchableByID{
		tribes: tribes,
	}
}

func searchByID(haystack searchableByID, id int) int {
	low := 0
	high := haystack.Len() - 1

	for low <= high {
		median := (low + high) / 2

		if haystack.GetID(median) < id {
			low = median + 1
		} else {
			high = median - 1
		}
	}

	if low == haystack.Len() || haystack.GetID(low) != id {
		return 0
	}

	return low
}
