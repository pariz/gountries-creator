package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Masterminds/glide/vendor/gopkg.in/yaml.v2"
	"github.com/codegangsta/cli"
	"github.com/pariz/gountries"
)

var (
	srcPath  string
	distPath string
)

func createFiles(c *cli.Context) {
	fmt.Println("Running: create")

	_, f, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}

	srcPath = filepath.Join(path.Dir(f), "src")
	distPath = filepath.Join(path.Dir(f), "dist")

	// Create dist folders if not present
	os.MkdirAll(filepath.Join(distPath, "yaml", "countries"), 0775)
	os.MkdirAll(filepath.Join(distPath, "yaml", "subdivisions"), 0775)
	os.MkdirAll(filepath.Join(distPath, "json", "countries"), 0775)
	os.MkdirAll(filepath.Join(distPath, "json", "subdivisions"), 0775)

	// Begin parsing and saving
	//
	var countries *[]gountries.Country
	var err error

	// Unrmarshal the large country.json file first and return a slice of countries
	if countries, err = populateCountriesFromJSON(); err != nil {
		fmt.Println("Could not parse JSON: " + err.Error())
		return
	}

	// Next, parse yaml files and return a slice with saveable data
	data := getSaveableData(countries)

	saveBytesToFiles(data)
}

func saveBytesToFiles(data map[string][]byte) {

	for path, bytes := range data {

		if err := ioutil.WriteFile(path, bytes, 0775); err != nil {
			fmt.Printf("Could not save data at %s: %s", path, err)
		}

	}

}

func getSaveableData(countries *[]gountries.Country) (saveData map[string][]byte) {

	saveData = make(map[string][]byte)

	for _, country := range *countries {

		var data map[string]interface{}

		ucAlpha2 := strings.ToUpper(country.Alpha2)

		// Populate from yaml
		//
		if bytes, err := ioutil.ReadFile(fmt.Sprintf("%s/countries/%s.yaml", srcPath, ucAlpha2)); err != nil {
			fmt.Printf("Error loading YAML: %s.yaml: %s\n", ucAlpha2, err)
			continue
		} else {
			if err = yaml.Unmarshal(bytes, &data); err != nil {
				fmt.Printf("Error parsing YAML: %s.yaml: %s\n", ucAlpha2, err)
				continue
			}
		}

		// Make sure we have data
		// The yaml package reflects all yaml data to map[interface{}]interface{}
		// as keys are arbirtrary in yaml and can be of any type.
		// Trying to cast the content to the given type proves if the data has
		// content or not.
		if data := data[ucAlpha2].(map[interface{}]interface{}); data != nil {

			//country.CountryCode = data["country_code"].(string)
			country.InternationalPrefix = data["international_prefix"].(string)
			country.Geo.Continent = data["continent"].(string)

			if data["eu_member"] != nil {
				country.EuMember = data["eu_member"].(bool)
			}

			// Coordinates
			country.MinLongitude = fVal(data["min_longitude"].(string))
			country.MinLatitude = fVal(data["min_latitude"].(string))
			country.MaxLongitude = fVal(data["max_longitude"].(string))
			country.MaxLatitude = fVal(data["max_latitude"].(string))

			country.Latitude = fVal(data["latitude_dec"].(string))
			country.Longitude = fVal(data["longitude_dec"].(string))

			country.LatitudeString = data["latitude"].(string)
			country.LongitudeString = data["longitude"].(string)

		}

		// Load subdivisions
		//

		if bytesSubd, err := ioutil.ReadFile(filepath.Join(srcPath, "subdivisions", fmt.Sprintf("%s.yaml", ucAlpha2))); err == nil {

			var subdMap = make(map[string]interface{})

			if err := yaml.Unmarshal(bytesSubd, &subdMap); err == nil {

				//spew.Dump(subdMap)
				subDivisions := []gountries.SubDivision{}

				for code, v := range subdMap {

					subd := gountries.SubDivision{}
					subd.CountryAlpha2 = country.Alpha2

					subd.Code = code

					if tmpData := v.(map[interface{}]interface{}); tmpData != nil {

						subd.Name = tmpData["name"].(string)
						names := tmpData["names"]

						switch names.(type) {
						case string:
							subd.Names = []string{names.(string)}
						case []interface{}:
							//spew.Dump(names)
							s := names.([]interface{})
							t := []string{}
							for _, v := range s {
								t = append(t, v.(string))
							}
							subd.Names = t
						}

						// Coordinates
						subd.MinLongitude, _ = tmpData["min_longitude"].(float64)
						subd.MinLatitude, _ = tmpData["min_latitude"].(float64)
						subd.MaxLongitude, _ = tmpData["max_longitude"].(float64)
						subd.MaxLatitude, _ = tmpData["max_latitude"].(float64)
						subd.Latitude, _ = tmpData["latitude"].(float64)
						subd.Longitude, _ = tmpData["longitude"].(float64)

					}

					subDivisions = append(subDivisions, subd)

				}

				// Create yaml
				if subdYamlBytes, err := yaml.Marshal(&subDivisions); err != nil {
					fmt.Printf("Could not marshal yaml data for %s: %s\n", subDivisions[0].CountryAlpha2, err)
				} else {

					saveData[filepath.Join(distPath, "yaml", "subdivisions", fmt.Sprintf("%s.yaml", strings.ToLower(subDivisions[0].CountryAlpha2)))] = subdYamlBytes
				}

				// Create json
				if subdJSONBytes, err := json.Marshal(&subDivisions); err != nil {
					fmt.Printf("Could not marshal json data for %s: %s\n", subDivisions[0].CountryAlpha2, err)
				} else {
					saveData[filepath.Join(distPath, "json", "subdivisions", fmt.Sprintf("%s.json", strings.ToLower(subDivisions[0].CountryAlpha2)))] = subdJSONBytes
				}

			} else {
				fmt.Printf("Could not parse subdivision YAML: %s.yaml: %s\n", ucAlpha2, err)
			}

		} else {
			fmt.Printf("Could not read subdivision YAML: %s.yaml: %s\n", ucAlpha2, err)
		}

		// Create yaml
		//

		if yamlBytes, err := yaml.Marshal(&country); err != nil {
			fmt.Printf("Could not marshal country YAML: %s.yaml: %s\n", country.Alpha2, err)
		} else {
			saveData[filepath.Join(distPath, "yaml", "countries", fmt.Sprintf("%s.yaml", strings.ToLower(country.Alpha2)))] = yamlBytes
		}

		// Create json
		//

		if jsonBytes, err := json.Marshal(&country); err != nil {
			fmt.Printf("Could not marshal country JSON: %s.json: %s\n", country.Alpha2, err)
		} else {
			saveData[filepath.Join(distPath, "json", "countries", fmt.Sprintf("%s.json", strings.ToLower(country.Alpha2)))] = jsonBytes

		}

	}
	return
}

func fVal(s string) (f float64) {

	var err error
	if f, err = strconv.ParseFloat(s, 64); err != nil {
		return 0
	}

	return f
}

func populateCountriesFromJSON() (countries *[]gountries.Country, err error) {

	var bytes []byte

	if bytes, err = ioutil.ReadFile(srcPath + "/countries.json"); err != nil {
		return
	}

	if err = json.Unmarshal(bytes, &countries); err != nil {
		return
	}

	// Uppercase translations
	//

	for ck, c := range *countries {

		for tk, t := range c.Translations {
			fmt.Println(ck, tk, t, strings.ToUpper(tk))

			c.Translations[strings.ToUpper(tk)] = t
			delete(c.Translations, tk)
		}

	}

	return
}
