package tasks

import (
	"github.com/tribalwarshelp/shared/tw/dataloader"
	"net/http"
	"time"

	"github.com/tribalwarshelp/shared/models"
)

func countPlayerVillages(villages []*models.Village) int {
	count := 0
	for _, village := range villages {
		if village.PlayerID != 0 {
			count++
		}
	}
	return count
}

func getDateDifferenceInDays(t1, t2 time.Time) int {
	hours := t1.Sub(t2).Hours()
	if hours == 0 {
		return 0
	}
	return int(hours / 24)
}

func calcPlayerDailyGrowth(diffInDays, points int) int {
	if diffInDays > 0 {
		return points / diffInDays
	}
	return 0
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}

func newDataloader(url string) dataloader.DataLoader {
	return dataloader.New(&dataloader.Config{
		BaseURL: url,
		Client:  newHTTPClient(),
	})
}

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

type ennoblementsSearchableByNewOwnerID struct {
	ennoblements []*models.Ennoblement
}

func (searchable ennoblementsSearchableByNewOwnerID) GetID(index int) int {
	return searchable.ennoblements[index].NewOwnerID
}

func (searchable ennoblementsSearchableByNewOwnerID) Len() int {
	return len(searchable.ennoblements)
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
		return -1
	}

	return low
}
