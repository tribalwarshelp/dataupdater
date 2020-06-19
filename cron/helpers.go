package cron

import (
	"compress/gzip"
	"encoding/csv"
	"encoding/xml"
	"io"
	"io/ioutil"
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

func getXML(url string, decode interface{}) error {
	resp, err := http.Get(url)
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
