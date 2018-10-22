package main

import (
	"SteamWebApiCrawler/bytechunk"
	"SteamWebApiCrawler/structs"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
)

func updateProducts(file *os.File, products *structs.Products) {
	var responsesNum int
	for reader := bytechunk.CreateByteReader(file); !reader.ReadNext(); {
		var data structs.ProductStoreDataDump
		err := proto.Unmarshal(reader.GetValue(), &data)
		if err != nil {
			fmt.Println("Error when unmarshaling response ", responsesNum, ": ", string(reader.GetValue()))
			continue
		}
		responsesNum++
		products.GetProduct(data.Id).StoreData = data.Data

		if responsesNum%10000 == 0 {
			fmt.Println("Parsed ", responsesNum)
		}
	}
	fmt.Println("Finished parsing. Parsed ", responsesNum)
}

func main() {
	products := structs.CreateProducts()
	products.LoadProducts("../data/products_with_stats.binary")

	os.Mkdir("data", 0777)
	files, err := ioutil.ReadDir("../dump")
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if strings.Index(file.Name(), "store_data") == 0 {
			fmt.Println(file.Name())
			file, err := os.Open("../dump/" + file.Name())
			if err != nil {
				fmt.Println("Could not open ", "../dump/"+file.Name(), " because: ", err)
				continue
			}
			updateProducts(file, products)
			defer file.Close()
		}
	}
	products.WriteProducts("../data/products_with_stats_and_market.binary")
}
