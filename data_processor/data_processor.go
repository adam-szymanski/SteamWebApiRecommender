package main

import (
	"SteamWebApiCrawler/structs"
	"fmt"
	"os"
	"sort"
)

func calculateProductPopularity(usersPerProduct map[uint64][]*structs.User, products *structs.Products, usersNum float32) {
	for productID := range usersPerProduct {
		product := products.GetProduct(productID)
		product.OwnersNum = uint64(len(usersPerProduct[productID]))
		product.FractionOfOwners = float32(product.OwnersNum) / usersNum
		if product.FractionOfOwners > 1 {
			fmt.Println(productID, product.OwnersNum, len(usersPerProduct))
		}
	}
}

func buildUsersPerProduct(users *structs.Users) map[uint64][]*structs.User {
	var usersNum int
	usersPerProduct := make(map[uint64][]*structs.User)
	for _, user := range users.Values {
		if len(user.GamesPlaytime) < 2 {
			continue
		}
		if len(user.GamesPlaytime) > 150 {
			continue
		}
		for productID := range user.GamesPlaytime {
			usersPerProduct[productID] = append(usersPerProduct[productID], user)
		}
		usersNum++
	}
	fmt.Println("User count: ", usersNum)
	return usersPerProduct
}

func calculateCoOwn(usersPerProduct map[uint64][]*structs.User, products *structs.Products, preferencesSize int, minOwners uint64) {
	fmt.Println("calculateCoOwn products num:", len(products.Values))
	productsProcessed := 0
	preferences := &structs.ProductsPreferences{
		Products: make(map[uint64]*structs.ProductPreferences),
	}
	products.Preferences.PreferencesMap[structs.CoownKey] = preferences

	revPreferences := &structs.ProductsPreferences{
		Products: make(map[uint64]*structs.ProductPreferences),
	}
	products.Preferences.PreferencesMap[structs.RevCoownKey] = revPreferences

	for productID := range usersPerProduct {
		product := products.GetProduct(productID)
		if product.GetOwnersNum() < minOwners {
			continue
		}
		productsCoOwn := make(map[uint64]uint64)
		for _, user := range usersPerProduct[productID] {
			for coOwned := range user.GamesPlaytime {
				productsCoOwn[coOwned]++
			}
		}
		for coOwnedID := range productsCoOwn {
			if products.GetProduct(coOwnedID).OwnersNum < minOwners {
				delete(productsCoOwn, coOwnedID)
			}
		}
		productsProcessed++
		if productsProcessed%1000 == 0 {
			fmt.Println("Processed ", productsProcessed)
		}

		keys := make([]uint64, 0, len(productsCoOwn))
		for key := range productsCoOwn {
			keys = append(keys, key)
		}

		// Regular co own.
		sort.Slice(keys, func(i, j int) bool {
			return productsCoOwn[keys[i]] > productsCoOwn[keys[j]]
		})
		preference := &structs.ProductPreferences{
			Preferences: make(map[uint64]float32),
			Percentiles: make([]float32, structs.DistributionSize),
		}
		ownersNum := float32(product.OwnersNum)
		preferences.Products[productID] = preference
		for i := 0; i < preferencesSize && i < len(keys); i++ {
			preference.Preferences[keys[i]] = float32(productsCoOwn[keys[i]]) / ownersNum
		}
		for i := 1; i < structs.DistributionSize+1; i++ {
			keyI := keys[len(keys)*i/(structs.DistributionSize+1)]
			preference.Percentiles[structs.DistributionSize-i] = float32(productsCoOwn[keyI]) / ownersNum
		}

		// Fraction of products belonging to current product owners
		sort.Slice(keys, func(i, j int) bool {
			keyI := keys[i]
			keyJ := keys[j]
			return (float32(productsCoOwn[keyI]) / float32(products.GetProduct(keyI).OwnersNum)) > (float32(productsCoOwn[keyJ]) / float32(products.GetProduct(keyJ).OwnersNum))
		})
		preference = &structs.ProductPreferences{
			Preferences: make(map[uint64]float32),
			Percentiles: make([]float32, structs.DistributionSize),
		}
		pref := preference.Preferences
		revPreferences.Products[productID] = preference
		for i := 0; len(pref) < preferencesSize && i < len(keys); i++ {
			keyI := keys[i]
			pref[keyI] = float32(productsCoOwn[keyI]) / float32(products.GetProduct(keyI).OwnersNum)
		}
		for i := 1; i < structs.DistributionSize+1; i++ {
			keyI := keys[len(keys)*i/(structs.DistributionSize+1)]
			preference.Percentiles[structs.DistributionSize-i] = float32(productsCoOwn[keyI]) / float32(products.GetProduct(keyI).OwnersNum)
		}

	}
	fmt.Println("Finished processing. Processed ", productsProcessed)
}

func calculatePercentileCoowns(products *structs.Products) {
	//TODO: use top percentile not from other preferences but recalculate actual co own.
	normPreferences := &structs.ProductsPreferences{
		Products: make(map[uint64]*structs.ProductPreferences),
	}
	products.Preferences.PreferencesMap[structs.NormCoownKey] = normPreferences

	normRevPreferences := &structs.ProductsPreferences{
		Products: make(map[uint64]*structs.ProductPreferences),
	}
	products.Preferences.PreferencesMap[structs.NormRevCoownKey] = normRevPreferences

	revPref := products.Preferences.PreferencesMap[structs.RevCoownKey].Products
	coownPref := products.Preferences.PreferencesMap[structs.CoownKey].Products

	fmt.Println("calculatePercentileCoowns products num:", len(products.Values))
	productsProcessed := 0
	for productID, preferences := range coownPref {
		productsProcessed++
		if productsProcessed%1000 == 0 {
			fmt.Println("Processed ", productsProcessed)
		}

		resultPreference := &structs.ProductPreferences{
			Preferences: make(map[uint64]float32),
		}
		prefMap := resultPreference.Preferences
		normPreferences.Products[productID] = resultPreference
		for prodID, val := range preferences.Preferences {
			prefMap[prodID] = revPref[prodID].GetPercentile(val)
		}
	}
	fmt.Println("Finished processing. Processed ", productsProcessed)

	fmt.Println("calculatePercentileRevCoowns products num:", len(products.Values))
	productsProcessed = 0
	for productID, preferences := range revPref {
		productsProcessed++
		if productsProcessed%1000 == 0 {
			fmt.Println("Processed ", productsProcessed)
		}

		resultPreference := &structs.ProductPreferences{
			Preferences: make(map[uint64]float32),
		}
		prefMap := resultPreference.Preferences
		normRevPreferences.Products[productID] = resultPreference
		for prodID, val := range preferences.Preferences {
			prefMap[prodID] = coownPref[prodID].GetPercentile(val)
		}
	}
	fmt.Println("Finished processing. Processed ", productsProcessed)
}

func calculateRelevance(products *structs.Products, preferencesSize int, minOwners uint64) {
	normPreferences := products.Preferences.PreferencesMap[structs.NormCoownKey].Products
	revPreferences := products.Preferences.PreferencesMap[structs.NormRevCoownKey].Products
	resultPreferences := &structs.ProductsPreferences{
		Products: make(map[uint64]*structs.ProductPreferences),
	}
	products.Preferences.PreferencesMap[structs.RelevanceKey] = resultPreferences

	for productID := range products.Values {
		product := products.GetProduct(productID)
		if product.GetOwnersNum() < minOwners {
			continue
		}

		pref, found := revPreferences[productID]
		if !found {
			continue
		}
		normPreferencesMap := normPreferences[productID].Preferences
		tempMap := make(map[uint64]float32)
		for id, val := range normPreferencesMap {
			revProb, found := pref.Preferences[id]
			if found {
				tempMap[id] = val * revProb
			}
		}

		keys := make([]uint64, 0, len(tempMap))
		for key := range tempMap {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			return tempMap[keys[i]] > tempMap[keys[j]]
		})

		result := &structs.ProductPreferences{
			Preferences: make(map[uint64]float32),
		}
		for i := 0; i < preferencesSize && i < len(keys); i++ {
			result.Preferences[keys[i]] = tempMap[keys[i]]
		}
		resultPreferences.Products[productID] = result
	}
}

func main() {
	users := structs.CreateUsers()
	products := structs.CreateProducts()

	users.LoadUsers("../data/users.binary", 1)

	usersPerProduct := buildUsersPerProduct(users)
	calculateProductPopularity(usersPerProduct, products, float32(len(users.Values)))
	calculateCoOwn(usersPerProduct, products, 1000, 50)
	calculatePercentileCoowns(products)
	calculateRelevance(products, 200, 50)

	os.Mkdir("../data", 0777)
	products.WriteProducts("../data/products_with_stats.binary")
}
