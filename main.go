package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
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
	Name     string
	Address  string
	Lat      float64
	Long     float64
	Selected []Store
}

type Store struct {
	Name     string
	Address  string
	Long     float64
	Lat      float64
	Comments string
	Selected []Person
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
		if p.Folders[i].XMLName.Local == "Folder" {
			for j := 0; j < len(p.Folders[i].Placemarks); j++ {
				ttype := GetType(p.Folders[i].Placemarks[j])
				if ttype == "person" {
					p.Persons = append(p.Persons, p.ParsePerson(p.Folders[i].Placemarks[j]))
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
	var s_name []string
	var s_address []string
	var s_comment []string
	var long []float64
	var lat []float64
	var p_name []string
	var p_address []string
	for i := 0; i < len(stores); i++ {
		for j := 0; j < len(persons); j++ {
			distance := GetDistance(stores[i].Lat, stores[i].Long, persons[j].Lat, persons[j].Long, "M")
			if distance <= limit {
				stores[i].Selected = append(stores[i].Selected, persons[j])
				persons[j].Selected = append(persons[j].Selected, stores[i])

				for p := 0; p < len(stores[i].Selected); p++ {
					s_name = append(s_name, stores[i].Name)
					s_address = append(s_address, stores[i].Address)
					s_comment = append(s_comment, stores[i].Comments)
					long = append(long, stores[i].Long)
					lat = append(lat, stores[i].Lat)
					p_name = append(p_name, stores[i].Selected[p].Name)
					p_address = append(p_address, stores[i].Selected[p].Address)
				}
			}
		}
	}

	CreateExcelFile(s_name, s_address, s_comment, long, lat, p_name, p_address)
}

func GetDistance(lat1 float64, lng1 float64, lat2 float64, lng2 float64, unit ...string) int {
	const PI float64 = 3.141592653589793

	radlat1 := float64(PI * lat1 / 180)
	radlat2 := float64(PI * lat2 / 180)

	theta := float64(lng1 - lng2)
	radtheta := float64(PI * theta / 180)

	dist := math.Sin(radlat1)*math.Sin(radlat2) + math.Cos(radlat1)*math.Cos(radlat2)*math.Cos(radtheta)

	if dist > 1 {
		dist = 1
	}

	dist = math.Acos(dist)
	dist = dist * 180 / PI
	dist = dist * 60 * 1.1515

	if len(unit) > 0 {
		if unit[0] == "K" {
			dist = dist * 1.609344
		} else if unit[0] == "N" {
			dist = dist * 0.8684
		} else if unit[0] == "M" {
			dist = dist * 1609.34
		}
	}

	return int(dist)
}

func CreateExcelFile(s_name []string, s_address []string,
	s_comment []string, long []float64,
	lat []float64, p_name []string, p_address []string) {
	sname := make(map[string]string)
	saddress := make(map[string]string)
	scomment := make(map[string]string)
	slon := make(map[string]float64)
	slat := make(map[string]float64)
	pname := make(map[string]string)
	paddress := make(map[string]string)
	for i := 0; i < len(s_name); i++ {
		index := i + 2
		iName := "A" + strconv.Itoa(index)
		iAddress := "B" + strconv.Itoa(index)
		iComment := "C" + strconv.Itoa(index)
		iLon := "D" + strconv.Itoa(index)
		iLat := "E" + strconv.Itoa(index)
		iPname := "F" + strconv.Itoa(index)
		iPaddress := "G" + strconv.Itoa(index)
		sname[iName] = s_name[i]
		saddress[iAddress] = s_address[i]
		scomment[iComment] = s_comment[i]
		slon[iLon] = long[i]
		slat[iLat] = lat[i]
		pname[iPname] = p_name[i]
		paddress[iPaddress] = p_address[i]
	}
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "Сеть")
	f.SetCellValue("Sheet1", "B1", "Адрес")
	f.SetCellValue("Sheet1", "C1", "Комментарий")
	f.SetCellValue("Sheet1", "D1", "Широта")
	f.SetCellValue("Sheet1", "E1", "Долгота")
	f.SetCellValue("Sheet1", "F1", "ФИО мерча в пределах ... метров")
	f.SetCellValue("Sheet1", "G1", "Адрес мерча")
	for k, v := range sname {
		f.SetCellValue("Sheet1", k, v)
	}
	for k, v := range saddress {
		f.SetCellValue("Sheet1", k, v)
	}
	for k, v := range scomment {
		f.SetCellValue("Sheet1", k, v)
	}
	for k, v := range slon {
		f.SetCellValue("Sheet1", k, v)
	}
	for k, v := range slat {
		f.SetCellValue("Sheet1", k, v)
	}
	for k, v := range pname {
		f.SetCellValue("Sheet1", k, v)
	}
	for k, v := range paddress {
		f.SetCellValue("Sheet1", k, v)
	}

	if err := f.SaveAs("/Users/bulahigor/goprojects/open/extract_data.xlsx"); err != nil {
		fmt.Println(err)
	}
}

func main() {

	var parser KmlParser

	p := parser.StartParse("/Users/bulahigor/goprojects/open/open_test2.kml")

	data := p.ParseFolder()

	CreateDataMap(data.Stores, data.Persons, 1000)
}
