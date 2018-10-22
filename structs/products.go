package structs

import (
	"SteamWebApiCrawler/bytechunk"
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"

	proto "github.com/golang/protobuf/proto"
)

// CoownKey Fraction of game owners having other game.
const CoownKey string = "coown"

// NormCoownKey
const NormCoownKey string = "norm_coown"

// RelevanceKey
const RelevanceKey string = "relevance"

// RevCoownKey
const RevCoownKey string = "rev_coown"

// NormRevCoownKey
const NormRevCoownKey string = "norm_rev_coown"

// DistributionSize is size of distribution
const DistributionSize int = 999

// Products contains and manages products.
type Products struct {
	Values      map[uint64]*Product
	Preferences PreferencesMap
}

// GetProduct returns product with given ID.
func (products *Products) GetProduct(productID uint64) *Product {
	product, isPresent := products.Values[productID]
	if !isPresent {
		product = &Product{
			Id: productID,
		}
		products.Values[productID] = product
	}
	return product
}

// GetProductName returns name of product with given ID.
func (products *Products) GetProductName(productID uint64) string {
	product, isPresent := products.Values[productID]
	if !isPresent {
		return "Unknown #" + strconv.FormatUint(productID, 10)
	}
	if product.StoreData == nil {
		return "Unnamed #" + strconv.FormatUint(productID, 10)
	}
	return product.StoreData.Name
}

// WriteProducts writes products in binary format.
func (products *Products) WriteProducts(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	err = binary.Write(f, binary.LittleEndian, int64(len(products.Values)))
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, product := range products.Values {
		bytes, err := proto.Marshal(product)
		if err != nil {
			fmt.Println(err)
			return
		}
		bytechunk.WriteBytes(f, bytes)
	}

	bytes, err := proto.Marshal(&products.Preferences)
	if err != nil {
		fmt.Println(err)
		return
	}
	bytechunk.WriteBytes(f, bytes)
}

// LoadProducts reads binary written products from given file.
func (products *Products) LoadProducts(filename string) {
	fmt.Println("Loading products.")
	f, err := os.Open(filename)
	if err != nil {
		fmt.Println("Could not read product file.")
		return
	}
	defer f.Close()

	var productsNum int64
	err = binary.Read(f, binary.LittleEndian, &productsNum)
	i := int64(0)
	for reader := bytechunk.CreateByteReader(f); i < productsNum; i++ {
		if err != nil {
			panic(err)
		}
		product := &Product{}
		err := proto.Unmarshal(reader.GetValue(), product)
		if err != nil {
			fmt.Println("Error when unmarshaling Product: ", reader.GetValue())
			continue
		}
		products.Values[product.Id] = product
		if i+1 < productsNum {
			reader.ReadNext()
		}
	}

	var buffer bytes.Buffer
	_, err = bytechunk.ReadBytes(f, &buffer)
	if err != nil {
		fmt.Println("Could not read preferences.")
		return
	}
	err = proto.Unmarshal(buffer.Bytes(), &products.Preferences)
	if err != nil {
		fmt.Println("Could not unmarshal preferences.")
		return
	}
	fmt.Println("Loaded ", len(products.Values), " products.")
}

// CreateProducts returns new map of products.
func CreateProducts() *Products {
	products := &Products{
		Values: make(map[uint64]*Product),
	}
	products.Preferences.PreferencesMap = make(map[string]*ProductsPreferences)
	return products
}

// GetPercentile returns value percentile.
func (preferences *ProductPreferences) GetPercentile(val float32) float32 {
	//TODO: change to bin search.
	for i := len(preferences.Percentiles) - 1; i >= 0; i-- {
		if val > preferences.Percentiles[i] {
			return float32(i+1) / float32(DistributionSize+1)
		}
	}
	return 0
}
