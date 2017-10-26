package main

import (
	"encoding/csv"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"flag"

	"fmt"

	log "github.com/sirupsen/logrus"
)

type banner struct {
	url     string
	shows   uint64
	catName []string
}

type category struct {
	name       string
	bannersPos []int
	shows      uint64
}

func loadBanners(fileName string) ([]banner, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	banners := make([]banner, 0, 1000)
	cr := csv.NewReader(file)
	cr.Comma = ';'

	for {
		cr.FieldsPerRecord = 0
		record, err := cr.Read()
		if err == io.EOF {
			return banners, nil
		}
		if err != nil {
			return nil, err
		}

		if len(record) < 3 {
			log.Errorf("bad record: %v, len = %d", record, len(record))
			continue
		}

		if _, err := url.Parse(record[0]); err != nil {
			log.Errorf("bad url %s in %v", record[0], record)
		}

		shows, err := strconv.ParseUint(record[1], 10, 0)
		if err != nil {
			log.Errorf("bad shows value: %s in %v", record[1], record)
			continue
		}

		banners = append(banners, banner{
			url:     record[0],
			shows:   shows,
			catName: record[2:],
		})
	}
}

func loadCategories(banners []banner) map[string]*category {
	result := make(map[string]*category)
	for i, b := range banners {
		for _, c := range b.catName {
			cat, ok := result[c]
			if !ok {
				cat = &category{
					bannersPos: make([]int, 0, 1),
					name:       c,
				}
			}

			cat.bannersPos = append(cat.bannersPos, i)
			cat.shows += b.shows
			result[c] = cat
		}
	}
	return result
}

func adHandler(banners []banner, cats map[string]*category) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Form == nil {
			err := r.ParseForm()
			if err != nil {
				log.Errorf("parse error %s", err)
				return
			}
		}
		userCats, ok := r.Form["category[]"]
		if !ok {
			log.Errorf("can't find parameter []category")
			return
		}

		b, err := getBanner(banners, cats, userCats)
		if err != nil {
			log.Errorf("choose banner error: %s", err)
			return
		}

		w.Write([]byte(fmt.Sprintf(`<html><body><p><img src="%s"></p></body></html>`, b)))
	}
}

func main() {
	var (
		fileName    string
		httpAddress string
	)

	flag.StringVar(&fileName, "storage", "", "path to banners description file")
	flag.StringVar(&httpAddress, "addr", ":8080", "http server address")
	flag.Parse()

	banners, err := loadBanners(fileName)
	if err != nil {
		log.Errorf("can't load banners: %s", err)
		os.Exit(1)
	}

	cats := loadCategories(banners)

	http.HandleFunc("/", adHandler(banners, cats))
	log.Fatal(http.ListenAndServe(httpAddress, nil))

}
