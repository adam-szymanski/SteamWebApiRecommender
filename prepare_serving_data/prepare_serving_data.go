package main

import (
	"SteamWebApiCrawler/bytechunk"
	"SteamWebApiCrawler/structs"
	"fmt"
	"os"
	"sort"

	"github.com/golang/protobuf/proto"
)

func main() {
	const maxSize = 50
	var products = structs.CreateProducts()
	products.LoadProducts("../data/products_with_stats_and_market.binary")
	fmt.Println("Converting products.")
	servingData := &structs.ServingData{
		PreferencesMap: make(map[string]*structs.ProductSortedPreferences),
		Products:       products.Values,
	}
	for preferenceName, preferences := range products.Preferences.PreferencesMap {
		fmt.Println("Converting ", preferenceName, " preference.")
		servingData.PreferencesMap[preferenceName] = &structs.ProductSortedPreferences{
			Products: make(map[uint64]*structs.ProductScoredList),
		}
		sortedPreferencesMap := servingData.PreferencesMap[preferenceName]
		for pid, productPreferences := range preferences.Products {
			result := make([]uint64, 0, len(productPreferences.Preferences))
			for preferenceID := range productPreferences.Preferences {
				result = append(result, preferenceID)
			}
			sort.Slice(result, func(i, j int) bool {
				return productPreferences.Preferences[result[i]] > productPreferences.Preferences[result[j]]
			})
			if len(result) > maxSize {
				result = result[:maxSize]
			}
			sortedPreferencesMap.Products[pid] = &structs.ProductScoredList{
				ProductIds: result,
			}
			sortedPreferencesMap.Products[pid].ProductIds = result

			scores := make([]float32, 0, len(result))
			for _, id := range result {
				scores = append(scores, productPreferences.Preferences[id])
			}
			sortedPreferencesMap.Products[pid].Scores = scores
		}
	}

	fmt.Println("Saving products.")
	f, err := os.Create("../data/serving_data.binary")
	if err != nil {
		fmt.Println("Could not write ../data/serving_data.binary.")
		return
	}
	bytes, err := proto.Marshal(servingData)
	if err != nil {
		fmt.Println(err)
		return
	}
	bytechunk.WriteBytes(f, bytes)
	defer f.Close()
}
