package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Data struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value"`
}

type Placemark struct {
	XMLName      xml.Name `xml:"Placemark"`
	Name         string   `xml:"name"`
	Address      string   `xml:"address"`
	ExtendedData []Data   `xml:"ExtendedData>Data"`
}

type Folder struct {
	XMLName    xml.Name    `xml:"Folder"`
	Name       string      `xml:"name"`
	Placemarks []Placemark `xml:"Placemark"`
}

type Document struct {
	XMLName xml.Name `xml:"Document"`
	Folders []Folder `xml:"Folder"`
}

type KML struct {
	XMLName  xml.Name `xml:"kml"`
	Document Document `xml:"Document"`
}

type Person struct {
	Name    string
	Address string
	Comment string
	// Coordinates []float64
	Lat  float64
	Long float64
	// Stores      []Store
	Type string
}

type Store struct {
	Name        string
	Address     string
	Comment     string
	Coordinates []float64
	Lat         float64
	Long        float64
	Persons     []Person
	Type        string
}

type KmlParser struct {
	Name    string
	Address string
	Comment string
	Type    string
	Persons []Person
	Stores  []Store
}

func GetType(pl Placemark) string {
	var ttype string
	for i := 0; i < len(pl.ExtendedData); i++ {
		if strings.ToLower(pl.ExtendedData[i].Name) == strings.ToLower("Адрес для информирования") {
			ttype = "person"
		} else if strings.ToLower(pl.ExtendedData[i].Name) == strings.ToLower("Широта") {
			ttype = "store"
		}
	}
	return ttype
}

func ParseFolder(fl Folder) ([]Person, []Store) {
	var person Person
	var store Store
	var persons []Person
	var stores []Store
	for i := 0; i < len(fl.Placemarks); i++ {
		ttype := GetType(fl.Placemarks[i])
		if ttype != "" {

			ParsePlacemark(fl.Placemarks[i], ttype, &person, &store)

			// fmt.Println(store)
			stores = append(stores, store)
			persons = append(persons, person)
		}
	}
	return persons, stores
}

func GetCoordinatesByAddress(address string) []string {
	uri := "https://geocode-maps.yandex.ru/1.x"
	yandex_api := "ea85fc44-fa6f-41db-8dd6-f968590a02fc"
	req, err := url.Parse(uri)
	if err != nil {
		fmt.Println(err)
	}

	params := url.Values{}
	params.Add("apikey", yandex_api)
	params.Add("format", "json")
	params.Add("geocode", address)
	req.RawQuery = params.Encode()

	resp, err := http.Get(req.String())

	if err != nil {
		fmt.Println(err.Error)
	}
	defer resp.Body.Close()
	body, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		fmt.Println(err2)
	}
	var f interface{}

	json.Unmarshal(body, &f)

	result := f.(map[string]interface{})
	response := result["response"].(map[string]interface{})
	GeoObjectCollection := response["GeoObjectCollection"].(map[string]interface{})
	featureMember := GeoObjectCollection["featureMember"].([]interface{})
	var coordinates []string
	for _, result := range featureMember {
		tmp := result.(map[string]interface{})
		GeoObject := tmp["GeoObject"].(map[string]interface{})
		Point := GeoObject["Point"].(map[string]interface{})
		Pos := Point["pos"].(string)
		coordinates = strings.Split(Pos, " ")
	}
	// fmt.Println(coordinates)
	return coordinates
}

func ParsePlacemark(pm Placemark, ttype string, p *Person, s *Store) {
	// Nname := ""
	var address string
	coordinates := make(map[string]float64)
	// Ccomment := ""
	// fmt.Println(Nname)
	// fmt.Println(Ccomment)
	// fmt.Println(pm.Address)

	// if pm.Name != "" {
	// 	Nname = pm.Name
	// }
	if pm.Address != "" {
		address = pm.Address
	}

	if ttype == "person" {
		if address != "" {
			coord := GetCoordinatesByAddress(address)
			if len(coord) > 1 {
				if long, err := strconv.ParseFloat(coord[0], 64); err == nil {
					coordinates["long"] = long
				}
				if lat, err := strconv.ParseFloat(coord[1], 64); err == nil {
					coordinates["lat"] = lat
				}
			}
		}
	} else if ttype == "store" {
		for i := 0; i < len(pm.ExtendedData); i++ {
			if strings.ToLower(pm.ExtendedData[i].Name) == strings.ToLower("Широта") {
				coord := pm.ExtendedData[i].Value
				if coord != "" {
					coordinate := strings.Split(coord, ", ")
					if len(coordinate) > 1 && len(coordinate[1]) > 0 && len(coordinate[0]) > 0 {
						if long, err := strconv.ParseFloat(coordinate[0], 64); err == nil {
							coordinates["long"] = long
						}
						if lat, err := strconv.ParseFloat(coordinate[1], 64); err == nil {
							coordinates["lat"] = lat
						}
						if address == "" {
							address = coord
						}
					}
				}
			} else if strings.ToLower(pm.ExtendedData[i].Name) == strings.ToLower("Комментарий") {
				// Ccomment = pm.ExtendedData[i].Value
			}
		}
	}

	if len(coordinates) > 0 {
		if ttype == "person" {
			// var p Person
			// defer p.CreatePerson(name, address, coordinates["long"], coordinates["lat"])
			// nam := "Igor"
			// defer HelloWorld(nam)

			// p.Name = name
			// p.Address = address
			// p.Long = coordinates["long"]
			// p.Lat = coordinates["lat"]
			// p.Type = "person"
		} else if ttype == "store" {
			// var s Store
			// defer s.CreateStore(name, address, coordinates["long"], coordinates["lat"], comment)
			// s.Name = name
			// s.Address = address
			// s.Comment = comment
			// s.Long = coordinates["long"]
			// s.Lat = coordinates["lat"]
			// s.Type = "store"
		}
	}

	// if ttype == "person" {
	// 	defer CreatePerson(name, address, coordinates["long"], coordinates["lat"])
	// } else if ttype == "store" {
	// 	defer CreateStore(name, address, coordinates["long"], coordinates["lat"], comment)
	// }

}

func StartParse() {

}

func main() {
	xmlFile, err := os.Open("/Users/bulahigor/goprojects/open/open_test2.kml")

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Successfully opened open_test2.kml")

	defer xmlFile.Close()

	byteValue, _ := ioutil.ReadAll(xmlFile)

	var document KML
	var parser KmlParser

	xml.Unmarshal(byteValue, &document)

	// dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(dir)

	for i := 0; i < len(document.Document.Folders); i++ {
		parser.Persons, parser.Stores = ParseFolder(document.Document.Folders[i])
	}

}

// env GOOS=linux GOARCH=amd64 go build -v github.com/constabulary/gb/cmd/gb
