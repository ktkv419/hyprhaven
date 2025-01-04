package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/reujab/wallpaper"
	"github.com/shirou/gopsutil/process"
)

func TerminateProcess(name string) error {
	processes, err := process.Processes()
	currentPid := os.Getpid()
	if err != nil {
		return err
	}
	for _, p := range processes {
		n, err := p.Name()
		if err != nil {
			return err
		}
		if n == name && int(p.Pid) != currentPid {
			return p.Terminate()
		}
	}
	return fmt.Errorf("process not found")
}

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

type RequestParams struct {
	SORTING    string
	CATEGORIES string
	Q          string
	PURITY     string
	ATLEAST    string
	RATIOS     string
	APIKEY     string
}

func setWallpaper() {
	PAGE_QUERY := 5

	REQUEST_PARAMS := RequestParams{
		SORTING:    "favorites",
		CATEGORIES: "010",
		Q:          "",
		PURITY:     "100",
		ATLEAST:    "1920x1080",
		RATIOS:     "landscape",
		// APIKEY:     "xxx",
	}

	initReqUrl, err := url.Parse("https://wallhaven.cc/api/v1/search")

	initReqParams := url.Values{}

	val := reflect.ValueOf(REQUEST_PARAMS)
	typ := reflect.TypeOf(REQUEST_PARAMS)

	for i := 0; i < val.NumField(); i++ {
		initReqParams.Add(strings.ToLower(typ.Field(i).Name), val.Field(i).String())
	}

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

	perPageInt, _ := strconv.Atoi(initApiResponse.Meta.PerPage)

	wallpaperPages := initApiResponse.Meta.Total/perPageInt + 1

	var wallpaperPage int
	var wallpaperPosition int

	if wallpaperPages < PAGE_QUERY {
		if initApiResponse.Meta.Total <= perPageInt {
			wallpaperPage = 1
			wallpaperPosition = rand.Intn(initApiResponse.Meta.Total)
		} else {
			lastPage := (initApiResponse.Meta.Total + perPageInt - 1) / perPageInt
			wallpaperPage = rand.Intn(lastPage) + 1
			if wallpaperPage == lastPage {
				wallpaperPosition = rand.Intn(initApiResponse.Meta.Total % perPageInt)
			} else {
				wallpaperPosition = rand.Intn(perPageInt)
			}
		}
	} else {
		//  BUG: on popular requests fetches wrong page
		wallpaperPage = rand.Intn(PAGE_QUERY) + 1
		wallpaperPosition = rand.Intn(perPageInt)
	}

	wallpaperReqUrl, _ := url.Parse("https://wallhaven.cc/api/v1/search")

	wallpaperReqParams := url.Values{}

	for i := 0; i < val.NumField(); i++ {
		wallpaperReqParams.Add(strings.ToLower(typ.Field(i).Name), val.Field(i).String())
	}

	wallpaperReqParams.Add("page", strconv.Itoa(wallpaperPage))

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

	wallpaper.SetMode(wallpaper.Crop)
	err = wallpaper.SetFromURL(wallpaperUrl)

	if err != nil {
		log.Fatalf("Error setting wallpaper: %v", err)
	}

}

func main() {
	TerminateProcess("hyprhaven.exe")
	for {
		setWallpaper()
		time.Sleep(15 * time.Minute)
	}
}
