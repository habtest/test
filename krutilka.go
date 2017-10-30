package main

import (
	"errors"
	"math/rand"
	"sort"
	"sync/atomic"
)

// choose wighted random position
func choosePos(weights []int) int {
	if len(weights) == 0 {
		return -1
	}

	if weights[len(weights)-1] == 0 {
		return -1
	}

	val := rand.Int31n(int32(weights[len(weights)-1]))
	return sort.Search(len(weights), func(i int) bool { return weights[i] >= int(val) })
}

func chooseCategory(cats map[string]*category, userCats []string) string {
	totalShows := 0
	catShows := make([]int, 0, len(cats))
	catNames := make([]string, 0, len(cats))

	for _, uCat := range userCats {
		if cat, ok := cats[uCat]; ok {
			shows := atomic.LoadUint64(&cat.shows)
			if shows == 0 {
				continue
			}
			totalShows += int(shows)
			catShows = append(catShows, totalShows)
			catNames = append(catNames, cat.name)
		}
	}
	pos := choosePos(catShows)
	if pos < 0 {
		return ""
	}
	return catNames[pos]
}

func chooseBanner(banners []banner, foundPos []int) (*banner, error) {
	totalShows := 0
	banShows := make([]int, 0, len(foundPos))
	banPos := make([]int, 0, len(foundPos))

	for _, pos := range foundPos {
		shows := atomic.LoadUint64(&banners[pos].shows)
		if shows == 0 {
			continue
		}
		totalShows += int(shows)
		banShows = append(banShows, totalShows)
		banPos = append(banPos, pos)
	}

	i := choosePos(banShows)
	if i < 0 {
		return nil, errors.New("can't find banner")
	}

	return &banners[banPos[i]], nil
}

func getBanner(banners []banner, cats map[string]*category, userCats []string) (string, error) {

	for {
		// choose category
		catName := chooseCategory(cats, userCats)
		if catName == "" {
			return "", errors.New("can't find category")
		}

		// choose banner from this category
		banner, err := chooseBanner(banners, cats[catName].bannersPos)
		if err != nil {
			// another goroutine got all banners from this category, try to choose another one
			continue
		}

		// try to decrement shows count for banner
		curShows := atomic.AddUint64(&banner.shows, ^uint64(0))
		if int64(curShows) < 0 {
			atomic.AddUint64(&banner.shows, 1)
			continue
		}

		// decrement shows count for this banner categories
		for _, catName = range banner.catName {
			if cat, ok := cats[catName]; ok {
				atomic.AddUint64(&cat.shows, ^uint64(0))
			}
		}
		return banner.url, nil
	}
}
