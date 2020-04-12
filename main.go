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

	"github.com/Luxurioust/excelize"
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
	Lat     float64
	Long    float64
}

type Store struct {
	Name     string
	Address  string
	Long     float64
	Lat      float64
	Comments string
}

type KmlParser struct {
	File    string
	Name    string
	Address string
	Comment string
	Type    string
	Folders []Folder
	Persons []Person
	Stores  []Store
}

func (p KmlParser) StartParse(file string) KmlParser {
	xmlFile, err := os.Open(file)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Successfully opened open_test2.kml")

	defer xmlFile.Close()

	byteValue, _ := ioutil.ReadAll(xmlFile)

	var document KML

	xml.Unmarshal(byteValue, &document)

	for i := 0; i < len(document.Document.Folders); i++ {
		p.Folders = append(p.Folders, document.Document.Folders[i])
	}

	return p
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
	return coordinates
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

func (p KmlParser) ParseFolder() KmlParser {
	for i := 0; i < len(p.Folders); i++ {
		// fmt.Printf("T%\n", p.Folders[0])
		// fmt.Println(p.Folders[i].XMLName.Local)
		if p.Folders[i].XMLName.Local == "Folder" {
			for j := 0; j < len(p.Folders[i].Placemarks); j++ {
				// fmt.Println(p.Folders[i].Placemarks[j])
				ttype := GetType(p.Folders[i].Placemarks[j])
				if ttype == "person" {
					// fmt.Printf("T%\n", p.Folders[i].Placemarks[j])
					// res := p.Folders[i].Placemarks[j]
					p.Persons = append(p.Persons, p.ParsePerson(p.Folders[i].Placemarks[j]))
					// fmt.Println(p)
				} else if ttype == "store" {
					p.Stores = append(p.Stores, p.ParseStore(p.Folders[i].Placemarks[j]))
				}
			}
		}
	}

	return p
}

func (p KmlParser) ParsePerson(pm Placemark) Person {
	var Name string
	Coordinates := make(map[string]float64)

	if pm.Name != "" {
		Name = pm.Name
	}
	if pm.Address != "" {
		coord := GetCoordinatesByAddress(pm.Address)
		if len(coord) > 1 {
			if long, err := strconv.ParseFloat(coord[0], 64); err == nil {
				Coordinates["long"] = long
			}
			if lat, err := strconv.ParseFloat(coord[1], 64); err == nil {
				Coordinates["lat"] = lat
			}
		}
	}

	if len(Coordinates) > 0 {
		pr := Person{Name: Name, Address: pm.Address, Long: Coordinates["long"], Lat: Coordinates["lat"]}
		return pr
	}
	return Person{}
}

func (p KmlParser) ParseStore(pm Placemark) Store {
	com := ""
	Coordinates := make(map[string]float64)
	for i := 0; i < len(pm.ExtendedData); i++ {
		if strings.ToLower(pm.ExtendedData[i].Name) == strings.ToLower("Широта") {
			coord := pm.ExtendedData[i].Value
			if coord != "" {
				c := strings.Split(coord, ", ")
				if len(c) > 1 && len(c[0]) > 0 && len(c[1]) > 0 {
					if long, err := strconv.ParseFloat(c[0], 64); err == nil {
						Coordinates["long"] = long
					}
					if lat, err := strconv.ParseFloat(c[1], 64); err == nil {
						Coordinates["lat"] = lat
					}
				}
				if pm.Address == "" {
					pm.Address = coord
				}
			}
		} else if strings.ToLower(pm.ExtendedData[i].Name) == strings.ToLower("Комментарий") {
			com = pm.ExtendedData[i].Value
		}
	}
	if len(Coordinates) > 0 {
		st := Store{Name: pm.Name, Address: pm.Address, Long: Coordinates["long"], Lat: Coordinates["lat"]}
		st.Comments = com
		return st
	}
	return Store{}
}

func CreateDataMap(stores []Store, persons []Person, limit int) {

}

func GetDistance(long1 float64, lat1 float64, long2 float64, lat2 float64) {
	uri := "https://api.routing.yandex.net/v1.0.0/route"
	yandex_api := "ea85fc44-fa6f-41db-8dd6-f968590a02fc"
	l1 := strconv.FormatFloat(long1, 'f', -1, 64)
	lt1 := strconv.FormatFloat(lat1, 'f', -1, 64)
	l2 := strconv.FormatFloat(long2, 'f', -1, 64)
	lt2 := strconv.FormatFloat(lat2, 'f', -1, 64)
	waypoints := l1 + "," + lt1 + "|" + l2 + "," + lt2
	// fmt.Println(waypoints)
	req, err := url.Parse(uri)
	if err != nil {
		fmt.Println(err)
	}
	params := url.Values{}
	params.Add("apikey", yandex_api)
	params.Add("format", "json")
	params.Add("waypoints", waypoints)
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
}

func CreateExcelFile(s_name string, s_address string,
	s_comment string, long float64,
	lat float64, p_name string, p_address string) {
	f := excelize.NewFile()
	// Create a new sheet.
	// index := f.NewSheet("Sheet2")
	// Set value of a cell.
	// f.SetCellValue("Sheet2", "A2", "Hello world.")
	f.SetCellValue("Sheet1", "B2", 100)
	// Set active sheet of the workbook.
	// f.SetActiveSheet(index)
	// Save xlsx file by the given path.
	if err := f.SaveAs("Book1.xlsx"); err != nil {
		fmt.Println(err)
	}
}

func main() {

	var parser KmlParser

	p := parser.StartParse("/Users/bulahigor/goprojects/open/open_test2.kml")

	data := p.ParseFolder()

	CreateDataMap(data.Stores, data.Persons, 1000)
	// CreateExcelFile()
	// fmt.Println(folder)
	GetDistance(55.734494627139355, 37.68191922355621, 55.733441295701056, 37.59027350593535)
}
