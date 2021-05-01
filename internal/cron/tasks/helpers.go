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

type playersSearchableByID struct {
	players []*models.Player
}

func (searchable playersSearchableByID) getID(index int) int {
	return searchable.players[index].ID
}

func (searchable playersSearchableByID) len() int {
	return len(searchable.players)
}

type tribesSearchableByID struct {
	tribes []*models.Tribe
}

func (searchable tribesSearchableByID) getID(index int) int {
	return searchable.tribes[index].ID
}

func (searchable tribesSearchableByID) len() int {
	return len(searchable.tribes)
}

type villagesSearchableByID struct {
	villages []*models.Village
}

func (searchable villagesSearchableByID) getID(index int) int {
	return searchable.villages[index].ID
}

func (searchable villagesSearchableByID) len() int {
	return len(searchable.villages)
}

type ennoblementsSearchableByNewOwnerID struct {
	ennoblements []*models.Ennoblement
}

func (searchable ennoblementsSearchableByNewOwnerID) getID(index int) int {
	return searchable.ennoblements[index].NewOwnerID
}

func (searchable ennoblementsSearchableByNewOwnerID) len() int {
	return len(searchable.ennoblements)
}

type searchableByID interface {
	getID(index int) int
	len() int
}

func searchByID(haystack searchableByID, id int) int {
	low := 0
	high := haystack.len() - 1

	for low <= high {
		median := (low + high) / 2

		if haystack.getID(median) < id {
			low = median + 1
		} else {
			high = median - 1
		}
	}

	if low == haystack.len() || haystack.getID(low) != id {
		return -1
	}

	return low
}
