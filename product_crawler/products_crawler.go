package main

import (
	"SteamWebApiCrawler/bytechunk"
	"SteamWebApiCrawler/structs"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
)

var titleRegexp = regexp.MustCompile("<b>Title:</b>[\n\t\r ]*([^<]*)<br>")
var tagRegexp = regexp.MustCompile("class=\"app_tag\"[^>]*>[\n\t\r]*([^\n\t\r]*)[\n\t\r]*</a>")
var banerRegexp = regexp.MustCompile("<img class=\"game_header_image_full\" src=\"([^\"]*)\">")
var genreRegexp = regexp.MustCompile("<a href=\"http://store.steampowered.com/genre/([^/]*)/\\?snr=1_5_9__408")
var flagsRegexp = regexp.MustCompile("<a class=\"name\" href=\"http://store.steampowered.com/search/\\?category2=[^>]*\">([^<]*)</a>")
var developerRegexp = regexp.MustCompile("<a[^?]*\\?developer[^>]*>([^<]*)")
var publisherRegexp = regexp.MustCompile("<b>Publisher:</b>\\s*(<a[^>]*>([^<]+)</a>,?\\s*)+\\s*<br>")
var releaseDateRegexp = regexp.MustCompile("<b>Release Date:</b> ([^<]*)<br>")
var reviewsRegexp = regexp.MustCompile("([0-9]+)% of the [0-9,]+ user reviews for")
var thumbnailsRegexp = regexp.MustCompile("highlight_strip_screenshot\" id=\"([^\"]*)")
var trailersRegexp = regexp.MustCompile("data-webm-source=\"([^\"]*)")

func produceIncompleteProduts(products *structs.Products, productsChan chan *structs.Product) {
	for _, product := range products.Values {
		if product.StoreData == nil || len(product.StoreData.Name) > 0 {
			productsChan <- product
		}
	}
	close(productsChan)
}

var httpClient = &http.Client{}

var requestsNumMutex = &sync.Mutex{}
var requestsNum = 0

func getRegexList(regexp *regexp.Regexp, text string, index int) ([]string, error) {
	regexMatch := regexp.FindAllStringSubmatch(text, -1)
	result := make([]string, 0, len(regexMatch))
	for _, match := range regexMatch {
		if index >= len(match) {
			fmt.Println("len")
			return nil, errors.New("index out of bound exception")
		}
		result = append(result, match[index])
	}
	return result, nil
}

func sendRequests(productsChan chan *structs.Product, storeDataChan chan *structs.ProductStoreDataDump, requestSendersWg *sync.WaitGroup) {
	for product := range productsChan {
		req, err := http.NewRequest("GET", "http://store.steampowered.com/app/"+strconv.FormatUint(product.GetId(), 10), nil)
		if err != nil {
			fmt.Println(err)
			continue
		}
		requestsNumMutex.Lock()
		requestsNum++
		if requestsNum%100 == 0 {
			fmt.Println(time.Now().Format("2006-01-02 15:04:05"), " requestsNum: ", requestsNum)
		}
		requestsNumMutex.Unlock()
		req.AddCookie(&http.Cookie{Name: "lastagecheckage", Value: "1-January-1987", Path: "/", Domain: "store.steampowered.com"})
		req.AddCookie(&http.Cookie{Name: "birthtime", Value: "536454001", Path: "/", Domain: "store.steampowered.com"})
		req.AddCookie(&http.Cookie{Name: "Steam_Language", Value: "english", Path: "/", Domain: "store.steampowered.com"})
		req.AddCookie(&http.Cookie{Name: "mature_content", Value: "1", Path: "/app/" + strconv.FormatUint(product.GetId(), 10), Domain: "store.steampowered.com"})
		req.AddCookie(&http.Cookie{Name: "app_impressions", Value: strconv.FormatUint(product.GetId(), 10) + "@1_4_4__100", Path: "/", Domain: "store.steampowered.com"})
		resp, err := httpClient.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode != 200 {
			fmt.Println(time.Now().Format("2006-01-02 15:04:05"), " StatusCode: ", resp.StatusCode, " URL: ", resp.Request.RequestURI)
			continue
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		bodyStr := string(body)
		if strings.Index(bodyStr, "<title>Welcome to Steam</title>") != -1 ||
			strings.Index(bodyStr, "<title>Site Error</title>") != -1 ||
			strings.Index(bodyStr, "<title>Steam Community") != -1 {
			continue
		}

		storeData := &structs.ProductStoreData{}

		nameMatch := titleRegexp.FindStringSubmatch(bodyStr)
		if len(nameMatch) > 1 {
			storeData.Name = nameMatch[1]
		} else {
			fmt.Println("Could not find name for ", product.Id, " with body:\n", bodyStr)
			os.Exit(1)
		}

		genre, err := getRegexList(genreRegexp, bodyStr, 1)
		if err != nil || len(genre) == 0 {
			fmt.Println("Could not find genre for ", product.Id)
		} else {
			storeData.Genre = genre
		}

		flags, err := getRegexList(flagsRegexp, bodyStr, 1)
		if err != nil || len(flags) == 0 {
			fmt.Println("Could not find flags for ", product.Id)
		} else {
			storeData.Flags = flags
		}

		tags, err := getRegexList(tagRegexp, bodyStr, 1)
		if err != nil || len(tags) == 0 {
			fmt.Println("Could not find tags for ", product.Id)
		} else {
			storeData.Tags = tags
		}

		banerMatch := banerRegexp.FindStringSubmatch(bodyStr)
		if len(banerMatch) > 1 {
			storeData.BanerUrl = banerMatch[1]
		} else {
			fmt.Println("Could not find baner for ", product.Id, " with body:\n", bodyStr)
			os.Exit(1)
		}

		developerMatch, err := getRegexList(developerRegexp, bodyStr, 1)
		if err != nil || len(developerMatch) == 0 {
			fmt.Println("Could not find developer for ", product.Id)
		} else {
			storeData.Developer = developerMatch
		}

		publisherMatch, err := getRegexList(publisherRegexp, bodyStr, 1)
		if err != nil || len(publisherMatch) == 0 {
			fmt.Println("Could not find publisher for ", product.Id)
		} else {
			storeData.Publisher = publisherMatch
		}

		thumbnailsMatch, err := getRegexList(thumbnailsRegexp, bodyStr, 1)
		if err != nil || len(thumbnailsMatch) == 0 {
			fmt.Println("Could not find thumbnails for ", product.Id)
		} else {
			storeData.Thumbnails = thumbnailsMatch
		}

		trailersMatch, err := getRegexList(trailersRegexp, bodyStr, 1)
		if err != nil || len(trailersMatch) == 0 {
			fmt.Println("Could not find trailers for ", product.Id)
		} else {
			storeData.Trailers = trailersMatch
		}

		releaseMatch := releaseDateRegexp.FindStringSubmatch(bodyStr)
		if len(releaseMatch) > 1 {
			t, err := time.Parse("_2 Jan, 2006", releaseMatch[1])
			if err != nil {
				t, err = time.Parse("Jan, 2006", releaseMatch[1])
				if err != nil {
					fmt.Println("Could not parse release date for ", product.Id, " with date:", releaseMatch[1])
				} else {
					storeData.ReleaseDate = t.Unix()
				}
			} else {
				storeData.ReleaseDate = t.Unix()
			}
		} else {
			fmt.Println("Could not find release date for ", product.Id)
		}

		reviewsMatch := reviewsRegexp.FindStringSubmatch(bodyStr)
		if len(reviewsMatch) > 1 {
			val, err := strconv.Atoi(reviewsMatch[1])
			if err != nil {
				fmt.Println("Could not parse reviews for ", product.Id, " with string:\n", reviewsMatch[1])
			} else {
				storeData.Reviews = int32(val)
			}
		} else {
			fmt.Println("Could not find reviews for ", product.Id)
		}

		storeDataChan <- &structs.ProductStoreDataDump{
			Data:      storeData,
			Timestamp: time.Now().Unix(),
			Id:        product.Id,
		}
	}
	requestSendersWg.Done()
}

func writeDump(name string, storeDataChan chan *structs.ProductStoreDataDump, dumpWriterWg *sync.WaitGroup) {
	dumpWriterWg.Add(1)
	var responsesNum int
	os.Mkdir("dump", 0777)
	f, err := os.OpenFile("../dump/store_data_"+name+".binary", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for dumpData := range storeDataChan {
		responsesNum++
		bytes, err := proto.Marshal(dumpData)
		if err != nil {
			fmt.Println(err)
			continue
		}
		err = bytechunk.WriteBytes(f, bytes)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if responsesNum%1000 == 0 {
			fmt.Println(time.Now().Format("2006-01-02 15:04:05"), " Dump progress: responses: ", responsesNum)
		}
	}
	fmt.Println("Dump progress: responses: ", responsesNum)
	dumpWriterWg.Done()
}

func probeProducts(products *structs.Products, productsChan chan *structs.Product) {
	requestSendersWg := new(sync.WaitGroup)
	threadsNum := 10
	storeDataChan := make(chan *structs.ProductStoreDataDump)
	for i := 0; i < threadsNum; i = i + 1 {
		requestSendersWg.Add(1)
		go sendRequests(productsChan, storeDataChan, requestSendersWg)
	}

	dumpWriterWg := new(sync.WaitGroup)
	go writeDump(time.Now().Format("2006-01-02_15"), storeDataChan, dumpWriterWg)
	requestSendersWg.Wait()
	close(storeDataChan)
	dumpWriterWg.Wait()
}

func main() {
	products := structs.CreateProducts()
	products.LoadProducts("../data/products_with_stats.binary")

	productsChan := make(chan *structs.Product)
	go produceIncompleteProduts(products, productsChan)
	probeProducts(products, productsChan)

	//probeRange()
}
