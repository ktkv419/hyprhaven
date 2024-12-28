package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"

	"github.com/reujab/wallpaper"
)

type Meta struct {
	Total   int    `json:"total"`
	Page    int    `json:"page"`
	PerPage string `json:"per_page"`
}

type Image struct {
	ID   string `json:"id"`
	PATH string `json:"path"`
}

type Response struct {
	Data []Image `json:"data"`
	Meta Meta    `json:"meta"`
}

func main() {
	initReqUrl, _ := url.Parse("https://wallhaven.cc/api/v1/search")

	initReqParams := url.Values{}

	initReqParams.Add("sorting", "favorites") // Example query: "nature"
	initReqParams.Add("categories", "010")    // Example: page 1
	initReqParams.Add("purity", "100")

	initReqUrl.RawQuery = initReqParams.Encode()

	// fmt.Print(parsedURL)

	// if err == nil {
	// 	fmt.Print(err)
	// }

	fmt.Println(initReqUrl.String())
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

	wallpaperCount := rand.Intn(initApiResponse.Meta.Total)

	wallpaperPage := wallpaperCount / 24

	wallpaperPosition := rand.Intn(24)

	wallpaperReqUrl, _ := url.Parse("https://wallhaven.cc/api/v1/search")

	wallpaperReqParams := url.Values{}

	wallpaperReqParams.Add("sorting", "favorites")              // Example query: "nature"
	wallpaperReqParams.Add("categories", "010")                 // Example: page 1
	wallpaperReqParams.Add("page", strconv.Itoa(wallpaperPage)) // Example: page 1
	// params.Add("purity", "100")

	wallpaperReqUrl.RawQuery = initReqParams.Encode()

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

	wallpaperErr := wallpaper.SetFromURL(wallpaperUrl)

	if wallpaperErr != nil {
		fmt.Print(wallpaperErr)
	}
}
