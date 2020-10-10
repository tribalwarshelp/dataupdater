package cron

import (
	"compress/gzip"
	"encoding/csv"
	"encoding/xml"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/tribalwarshelp/shared/models"
)

var client = &http.Client{
	Timeout: 20 * time.Second,
}

func newCsvReader(r io.Reader) *csv.Reader {
	csvReader := csv.NewReader(r)
	csvReader.LazyQuotes = true
	return csvReader
}

func uncompressAndReadCsvLines(r io.Reader) ([][]string, error) {
	uncompressedStream, err := gzip.NewReader(r)
	if err != nil {
		return [][]string{}, err
	}
	defer uncompressedStream.Close()
	return newCsvReader(uncompressedStream).ReadAll()
}

func getCSVData(url string, compressed bool) ([][]string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if !compressed {
		return newCsvReader(resp.Body).ReadAll()
	}
	return uncompressAndReadCsvLines(resp.Body)
}

func getXML(url string, decode interface{}) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return xml.Unmarshal(bytes, decode)
}

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
	diff := t1.Sub(t2)
	return int(diff.Hours() / 24)
}

func calcPlayerDailyGrowth(diffInDays, points int) int {
	if diffInDays > 0 {
		return points / diffInDays
	}
	return 0
}
