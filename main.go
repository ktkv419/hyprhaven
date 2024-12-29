package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/reujab/wallpaper"
)

type Meta struct {
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

type Image struct {
	ID   string `json:"id"`
	PATH string `json:"path"`
}

type Response struct {
	Data []Image `json:"data"`
	Meta Meta    `json:"meta"`
}

type RequestParams struct {
	PAGE_QUERY int
	SORTING    string
	CATEGORIES string
	QUERY      string
	PURITY     string
}

func setWallpaper() {
	REQUEST_PARAMS := RequestParams{
		PAGE_QUERY: 5,
		SORTING:    "favorites",
		CATEGORIES: "010",
		QUERY:      "",
		PURITY:     "100",
	}

	initReqUrl, err := url.Parse("https://wallhaven.cc/api/v1/search")

	initReqParams := url.Values{}

	initReqParams.Add("sorting", REQUEST_PARAMS.SORTING)
	initReqParams.Add("categories", REQUEST_PARAMS.CATEGORIES)
	initReqParams.Add("q", REQUEST_PARAMS.QUERY)
	initReqParams.Add("purity", REQUEST_PARAMS.PURITY)

	initReqUrl.RawQuery = initReqParams.Encode()

	if err != nil {
		log.Fatalf("Error decoding JSON request: %v", err)
	}

	initRes, err := http.Get(initReqUrl.String())

	if err != nil {
		log.Fatalf("Error making GET request: %v", err)
	}

	defer initRes.Body.Close()

	// Check if the response status is OK
	if initRes.StatusCode != http.StatusOK {
		log.Fatalf("Error: received status code %d", initRes.StatusCode)
	}

	// Decode the JSON response
	var initApiResponse Response
	err = json.NewDecoder(initRes.Body).Decode(&initApiResponse)

	if err != nil {
		log.Fatalf("Error decoding JSON response: %v", err)
	}

	wallpaperPages := initApiResponse.Meta.Total/initApiResponse.Meta.PerPage + 1

	var wallpaperPage int
	var wallpaperPosition int

	if wallpaperPages < REQUEST_PARAMS.PAGE_QUERY {
		if initApiResponse.Meta.Total <= initApiResponse.Meta.PerPage {
			wallpaperPage = 1
			wallpaperPosition = rand.Intn(initApiResponse.Meta.Total)
		} else {
			lastPage := (initApiResponse.Meta.Total + initApiResponse.Meta.PerPage - 1) / initApiResponse.Meta.PerPage
			wallpaperPage = rand.Intn(lastPage) + 1
			if wallpaperPage == lastPage {
				wallpaperPosition = rand.Intn(initApiResponse.Meta.Total % initApiResponse.Meta.PerPage)
			} else {
				wallpaperPosition = rand.Intn(initApiResponse.Meta.PerPage)
			}
		}
	} else {
		//  BUG: on popular requests fetches wrong page
		wallpaperPage = rand.Intn(REQUEST_PARAMS.PAGE_QUERY) + 1
		wallpaperPosition = rand.Intn(initApiResponse.Meta.PerPage)
	}

	wallpaperReqUrl, _ := url.Parse("https://wallhaven.cc/api/v1/search")

	wallpaperReqParams := url.Values{}

	wallpaperReqParams.Add("sorting", REQUEST_PARAMS.SORTING)
	wallpaperReqParams.Add("categories", REQUEST_PARAMS.CATEGORIES)
	wallpaperReqParams.Add("q", REQUEST_PARAMS.QUERY)
	wallpaperReqParams.Add("page", strconv.Itoa(wallpaperPage))
	wallpaperReqParams.Add("purity", REQUEST_PARAMS.PURITY)

	wallpaperReqUrl.RawQuery = wallpaperReqParams.Encode()

	fmt.Println(wallpaperReqUrl.String())

	wallpaperRes, err := http.Get(wallpaperReqUrl.String())

	if err != nil {
		log.Fatalf("Error making GET request: %v", err)
	}
	defer wallpaperRes.Body.Close()

	// Check if the response status is OK
	if wallpaperRes.StatusCode != http.StatusOK {
		log.Fatalf("Error: received status code %d", wallpaperRes.StatusCode)
	}

	// Decode the JSON response
	var wallpaperApiResponse Response
	err = json.NewDecoder(wallpaperRes.Body).Decode(&wallpaperApiResponse)

	if err != nil {
		log.Fatalf("Error decoding JSON response: %v", err)
	}

	wallpaperUrl := wallpaperApiResponse.Data[wallpaperPosition].PATH

	err = wallpaper.SetFromURL(wallpaperUrl)

	if err != nil {
		log.Fatalf("Error setting wallpaper: %v", err)
	}

}

func main() {
	for {
		setWallpaper()
		time.Sleep(15 * time.Minute)
	}
}
