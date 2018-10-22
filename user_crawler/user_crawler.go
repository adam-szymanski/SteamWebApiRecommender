package main

import (
	"SteamWebApiCrawler/bytechunk"
	"SteamWebApiCrawler/structs"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
)

const apiKey = "YOUR_KEY"

func readOwnedGames(user uint64, steamWebAPIKey string) (*structs.UserHttpResponse, error) {
	resp, err := http.Get("http://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?include_played_free_games=1&key=" + steamWebAPIKey + "&steamid=" + strconv.FormatUint(user, 10) + "&format=json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	response := &structs.UserHttpResponse{
		UserId: user,
		HttpResponse: &structs.HttpResponse{
			ResponseCode: uint32(resp.StatusCode),
			Timestamp:    time.Now().UnixNano(),
			Body:         body,
		},
	}
	return response, nil
}

func produceUserIds(rangeStart uint64, rangeEnd uint64, userIds chan uint64) {
	for i := rangeStart; i < rangeEnd; i = i + 1 {
		userIds <- i
	}
	close(userIds)
}

func produceRandomUserIds(rangeStart uint64, rangeEnd uint64, idsNum uint64, userIds chan uint64) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	diff := rangeEnd - rangeStart
	for i := uint64(1); i < idsNum; i = i + 1 {
		userIds <- rangeStart + (r.Uint64() % diff)
	}
	close(userIds)
}

func sendRequests(userIds chan uint64, requestSendersWg *sync.WaitGroup, responses chan *structs.UserHttpResponse) {
	for userID := range userIds {
		response, err := readOwnedGames(userID, apiKey)
		if err != nil {
			fmt.Println(err)
		} else {
			responses <- response
		}
	}
	requestSendersWg.Done()
}

func writeDump(name string, responses chan *structs.UserHttpResponse, dumpWriterWg *sync.WaitGroup) {
	dumpWriterWg.Add(1)
	var responsesNum int
	var responsesOKResponseCode int
	var emptyResponses int
	os.Mkdir("../dump", 0777)
	f, err := os.OpenFile("../dump/get_owned_games_"+name+".binary", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for response := range responses {
		responsesNum++
		if response.GetHttpResponse().GetResponseCode() == 200 {
			responsesOKResponseCode++
		}
		bytes, err := proto.Marshal(response)
		if err != nil {
			fmt.Println(err)
			continue
		}
		err = bytechunk.WriteBytes(f, bytes)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if strings.Index(string(response.GetHttpResponse().Body), "game_count") == -1 {
			emptyResponses++
		}
		if responsesNum%10 == 0 {
			fmt.Println(time.Now().Format("2006-01-02 15:04:05"), " Dump progress: responses: ", responsesNum, " OK responses: ", responsesOKResponseCode, " empty responses: ", emptyResponses)
		}
	}
	fmt.Println("Dump progress: responses: ", responsesNum, " OK responses: ", responsesOKResponseCode, " empty responses: ", emptyResponses)
	dumpWriterWg.Done()
}

const origingID uint64 = 76561197960265728
const maxID uint64 = 76561198454900000
const alafierID uint64 = 76561198030912048

func probeRandomEveryHour() {
	for {
		go func() {
			userIds := make(chan uint64)
			go produceRandomUserIds(origingID, maxID, 3760, userIds)
			responses := make(chan *structs.UserHttpResponse)

			requestSendersWg := new(sync.WaitGroup)
			coresNum := 100
			for i := 0; i < coresNum; i = i + 1 {
				requestSendersWg.Add(1)
				go sendRequests(userIds, requestSendersWg, responses)
			}
			dumpWriterWg := new(sync.WaitGroup)
			go writeDump(time.Now().Format("2006-01-02_15"), responses, dumpWriterWg)
			requestSendersWg.Wait()
			close(responses)
			dumpWriterWg.Wait()
		}()
		time.Sleep(time.Hour)
	}
}

func probeRange() {
	userIds := make(chan uint64)
	var rangeStart uint64 = 76561198031110448
	rangeEnd := rangeStart + 90000
	go produceUserIds(rangeStart, rangeEnd, userIds)
	responses := make(chan *structs.UserHttpResponse)

	requestSendersWg := new(sync.WaitGroup)
	coresNum := 1
	for i := 0; i < coresNum; i = i + 1 {
		requestSendersWg.Add(1)
		go sendRequests(userIds, requestSendersWg, responses)
	}
	dumpWriterWg := new(sync.WaitGroup)
	go writeDump(time.Now().Format("2006-01-02")+"_"+strconv.FormatUint(rangeStart, 10)+"_"+strconv.FormatUint(rangeEnd, 10), responses, dumpWriterWg)
	requestSendersWg.Wait()
	close(responses)
	dumpWriterWg.Wait()
}

func main() {
	probeRandomEveryHour()
}
