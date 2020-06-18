package cron

import (
	"compress/gzip"
	"encoding/csv"
	"io"
	"net/http"
)

func uncompressAndGetCsvLines(r io.Reader) ([][]string, error) {
	uncompressedStream, err := gzip.NewReader(r)
	if err != nil {
		return [][]string{}, err
	}
	defer uncompressedStream.Close()
	return csv.NewReader(uncompressedStream).ReadAll()
}

func getCSVData(url string, compressed bool) ([][]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if !compressed {
		return csv.NewReader(resp.Body).ReadAll()
	}
	return uncompressAndGetCsvLines(resp.Body)
}
