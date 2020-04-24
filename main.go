package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/Luxurioust/excelize"
)

/*
Data Структура Data
*/
type Data struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value"`
}

/*
Placemark Структура Placemark
*/
type Placemark struct {
	XMLName      xml.Name `xml:"Placemark"`
	Name         string   `xml:"name"`
	Address      string   `xml:"address"`
	ExtendedData []Data   `xml:"ExtendedData>Data"`
}

/*
Folder Структура Folder
*/
type Folder struct {
	XMLName    xml.Name    `xml:"Folder"`
	Name       string      `xml:"name"`
	Placemarks []Placemark `xml:"Placemark"`
}

/*
Document Структура Document
*/
type Document struct {
	XMLName xml.Name `xml:"Document"`
	Folders []Folder `xml:"Folder"`
}

/*
KML Структура kml файла
*/
type KML struct {
	XMLName  xml.Name `xml:"kml"`
	Document Document `xml:"Document"`
}

/*
Person Структура мерчендайзера
*/
type Person struct {
	Name     string
	Address  string
	Lat      float64
	Long     float64
	Selected []Store // TODO: Сделать список доступных ТТ уникальным
}

/*
Store Структура магазина
*/
type Store struct {
	Name     string
	Address  string
	Long     float64
	Lat      float64
	Comments string
	Selected []Person // TODO: Сделать список доступных мерчей уникальным
}

/*
KmlParser Структура парсера
*/
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

/*
StartParse Начало парсинга
*/
func (p KmlParser) StartParse(file string) KmlParser {
	xmlFile, err := os.Open(file)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Successfuly opened file " + strings.Split(file, "/")[len(strings.Split(file, "/"))-1])

	defer xmlFile.Close()

	byteValue, _ := ioutil.ReadAll(xmlFile)

	var document KML

	xml.Unmarshal(byteValue, &document)
	fmt.Println("Parsing in progress...")
	for i := 0; i < len(document.Document.Folders); i++ {
		p.Folders = append(p.Folders, document.Document.Folders[i])
	}

	return p
}

func uniquePerson(intSlice []Person) []Person {
	keys := make(map[*Person]bool)
	list := []Person{}
	for _, entry := range intSlice {
		if _, value := keys[&entry]; !value {
			keys[&entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func uniqueStore(intSlice []Store) []Store {
	keys := make(map[*Store]bool)
	list := []Store{}
	for _, entry := range intSlice {
		if _, value := keys[&entry]; !value {
			keys[&entry] = true
			list = append(list, entry)
		}
	}
	return list
}

/*
GetCoordinatesByAddress Получение координат по адресу
*/
func GetCoordinatesByAddress(address string) []string {
	uri := "https://geocode-maps.yandex.ru/1.x"
	yandexAPI := "ea85fc44-fa6f-41db-8dd6-f968590a02fc"
	req, err := url.Parse(uri)
	if err != nil {
		fmt.Println(err)
	}

	params := url.Values{}
	params.Add("apikey", yandexAPI)
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

/*
GetType Получить тип объекта
*/
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

/*
ParseFolder Парсинг блока ТТ или блока с мерчендайзерами
*/
func (p KmlParser) ParseFolder() KmlParser {
	for i := 0; i < len(p.Folders); i++ {
		if p.Folders[i].XMLName.Local == "Folder" {
			for j := 0; j < len(p.Folders[i].Placemarks); j++ {
				ttype := GetType(p.Folders[i].Placemarks[j])
				if ttype == "person" {
					person := p.ParsePerson(p.Folders[i].Placemarks[j])
					if person.Name != "" {
						p.Persons = append(p.Persons, person)
					}
					// TODO: Добавить определение города мерча
				} else if ttype == "store" {
					store := p.ParseStore(p.Folders[i].Placemarks[j])
					if store.Name != "" {
						p.Stores = append(p.Stores, store)
					}
					// TODO: Добавить определение города магазина
				}
			}
		}
	}

	return p
}

/*
ParsePerson Парсинг типа Person и создание объекта
*/
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

/*
ParseStore Парсинг типа Store и создание объекта
*/
func (p KmlParser) ParseStore(pm Placemark) Store {
	// TODO: Добавить проверку отсутсвия адреса и добавление адреса из блока ExtentionData
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

/*
CreateDataMap Подготовка данных, рассчет расстояния
*/
func CreateDataMap(stores []Store, persons []Person, limit int) {
	if limit == 0 {
		limit = 1000
	}
	var sName []string
	var sAddress []string
	var sComment []string
	var long []float64
	var lat []float64
	var pName []string
	var pAddress []string
	for i := 0; i < len(stores); i++ {
		for j := 0; j < len(persons); j++ {
			distance := GetDistance(stores[i].Lat, stores[i].Long, persons[j].Long, persons[j].Lat, "M")
			// TODO: Добавить возможность задавать дистанцию от и до
			if distance <= limit {
				stores[i].Selected = append(stores[i].Selected, persons[j])
				persons[j].Selected = append(persons[j].Selected, stores[i])
				// TODO: Добавить обработку магазинов, у которых нет доступных мерчей поблизости
				for p := 0; p < len(stores[i].Selected); p++ {
					sName = append(sName, stores[i].Name)
					sAddress = append(sAddress, stores[i].Address)
					sComment = append(sComment, stores[i].Comments)
					long = append(long, stores[i].Long)
					lat = append(lat, stores[i].Lat)
					pName = append(pName, stores[i].Selected[p].Name)
					pAddress = append(pAddress, stores[i].Selected[p].Address)
				}
			}
		}
	}
	// fmt.Println(len(stores))
	// for _, k := range stores {
	// 	fmt.Println("====", k, "\n")
	// }
	// uniqueStore(stores)
	// fmt.Println(len(stores))
	// for _, k := range stores {
	// 	fmt.Println("====", k, "\n")
	// }

	CreateExcelFile(sName, sAddress, sComment, long, lat, pName, pAddress)
}

/*
GetDistance Расчет дистанции между объектами
*/
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

/*
CreateExcelFile Создание файла excel и добавление данных
*/
func CreateExcelFile(sName []string, sAddress []string,
	sComment []string, long []float64,
	lat []float64, pName []string, pAddress []string) {
	sname := make(map[string]string)
	saddress := make(map[string]string)
	scomment := make(map[string]string)
	slon := make(map[string]float64)
	slat := make(map[string]float64)
	pname := make(map[string]string)
	paddress := make(map[string]string)
	for i := 0; i < len(sName); i++ {
		index := i + 2
		iName := "A" + strconv.Itoa(index)
		iAddress := "B" + strconv.Itoa(index)
		iComment := "C" + strconv.Itoa(index)
		iLon := "D" + strconv.Itoa(index)
		iLat := "E" + strconv.Itoa(index)
		iPname := "F" + strconv.Itoa(index)
		iPaddress := "G" + strconv.Itoa(index)
		sname[iName] = sName[i]
		saddress[iAddress] = sAddress[i]
		scomment[iComment] = sComment[i]
		slon[iLon] = long[i]
		slat[iLat] = lat[i]
		pname[iPname] = pName[i]
		paddress[iPaddress] = pAddress[i]
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

	dir, err := os.Getwd()
	path := dir + string(os.PathSeparator) + "export_data.xlsx"
	if err != nil {
		log.Fatal(err)
	}

	if err := f.SaveAs(path); err != nil {
		fmt.Println(err)
	}
}

func main() {
	fileName := "open.kml"
	dir, err := os.Getwd()
	path := dir + string(os.PathSeparator) + fileName
	if err != nil {
		log.Fatal(err)
	}

	var parser KmlParser
	// TODO: Добавить установку дистанции при компилировании
	p := parser.StartParse(path)

	data := p.ParseFolder()

	CreateDataMap(data.Stores, data.Persons, 1000)
	fmt.Println("Operation completed successfuly. Excel file was created.")
}
