package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
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
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

// UnmarshalJSON allows us to customize how the `PerPage` field is unmarshalled
func (m *Meta) UnmarshalJSON(data []byte) error {
	var temp struct {
		Total   int         `json:"total"`
		Page    int         `json:"page"`
		PerPage interface{} `json:"per_page"`
	}

	// First, unmarshal into the temporary struct
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Assign values to the Meta struct
	m.Total = temp.Total
	m.Page = temp.Page

	// Handle the `perPage` value dynamically
	switch v := temp.PerPage.(type) {
	case float64: // If it's a float64 (which is how numbers are unmarshalled in JSON)
		m.PerPage = int(v)
	case string:
		// If it's a string, try to convert it to an integer
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid perPage value: %v", v)
		}
		m.PerPage = parsed
	default:
		return fmt.Errorf("unsupported perPage type: %T", v)
	}

	return nil
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

func setWallpaper(REQUEST_PARAMS RequestParams, PAGE_QUERY int) {

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

	wallpaperPages := initApiResponse.Meta.Total/initApiResponse.Meta.PerPage + 1

	var wallpaperPage int
	var wallpaperPosition int

	if wallpaperPages < PAGE_QUERY {
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
		wallpaperPage = rand.Intn(PAGE_QUERY) + 1
		wallpaperPosition = rand.Intn(initApiResponse.Meta.PerPage)
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
	timerFlag := flag.Int("t", 15, "")
	PAGE_QUERY := flag.Int("pq", 5, "")

	sortingFlag := flag.String("s", "favorites", "")
	categoriesFlag := flag.String("c", "010", "")
	queryFlag := flag.String("q", "", "")
	purityFlag := flag.String("p", "100", "")
	atLeastFlag := flag.String("sz", "1920x1080", "")
	apikeyFlag := flag.String("key", "", "")

	flag.Parse()

	unsafeRegex, _ := regexp.Compile(`1$`)
	if unsafeRegex.MatchString(*purityFlag) && *apikeyFlag == "" {
		// Replace the last character with '0'
		*purityFlag = (*purityFlag)[:len(*purityFlag)-1] + "0"
	}

	REQUEST_PARAMS := RequestParams{
		SORTING:    *sortingFlag,
		CATEGORIES: *categoriesFlag,
		Q:          *queryFlag,
		PURITY:     *purityFlag,
		ATLEAST:    *atLeastFlag,
		APIKEY:     *apikeyFlag,
		RATIOS:     "landscape",
	}

	for {
		setWallpaper(REQUEST_PARAMS, *PAGE_QUERY)
		time.Sleep(time.Duration(*timerFlag) * time.Minute)
	}
}
