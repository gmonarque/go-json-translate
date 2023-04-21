package translator

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"

	"github.com/gmonarque/deepl-json/models"
)

func ReadJson(source_file string) (map[string]interface{}, error) {
	//Not a lot to say here, we unmarshal the json file to a map[string]interface{}
	//We can do that because we know that our json keys will be of type string,
	//but we don't know the type of the value so we declare a generic interface{}
	jsonFile, err := os.Open(source_file)

	defer jsonFile.Close()

	if err != nil {
		return nil, err
	}

	byteValue, _ := io.ReadAll(jsonFile)

	var result map[string]interface{}

	//json.Unmarshal is not really efficient, using json.Decode would be better
	//It doesn't really matter for small files, but this could be a problem for bigger files
	//json.Unmarshal decodes the whole file at once, and will not be efficient in case of a large file
	//json.Decode reads through the file incrementally, so the size of the file doesn't matter
	json.Unmarshal([]byte(byteValue), &result)

	return result, nil
}

func TranslateJson(config models.Config) (map[string]interface{}, error) {
	//This is where all the fun is happenning
	//First, we get the root keys of our json file
	keys := reflect.ValueOf(config.Source_data).MapKeys()

	//We then iterate on the keys
	for i := 0; i < len(keys); i++ {
		//For each key, we get the value (declared as elem below) associated
		//Once we have the value, we need to check its type because we don't know it yet
		//In order to do that, we call .(type) on the value and we switch over the type
		key := keys[i].Interface().(string)

		// Ignore keys specified in CLI parameters
		ignored := false
		for _, ignored_key := range config.Ignored_fields {
			if key == ignored_key {
				config.Translated_file[key] = config.Source_data[key]
				ignored = true
				break
			}
		}

		if ignored {
			continue
		}

		switch elem := config.Source_data[key].(type) {

		//If the value is also a map[string]interface{}, this means our json source file is nested
		//A "node" is declared, of type map[string]interface{}, and aw shit, here we go again
		//We call TranslateJson again (recursive), this time using elem as a source file, and the
		//newly created "node" to store the result.
		//Once something other than a map[string]interface{} has been reached, we store the result in translated_file
		//This allows us to keep the source json file nested structure
		case map[string]interface{}:
			config_node := config
			config_node.Source_data = elem
			config_node.Translated_file = map[string]interface{}{}
			config.Translated_file[key], _ = TranslateJson(config_node)
		//If we reach a list of values, we need to iterate over each value and translate it
		case []map[string]interface{}:
			for j := 0; j < len(elem); j++ {
				config_node := config
				config_node.Source_data = elem[j]
				config_node.Translated_file = map[string]interface{}{}
				config.Translated_file[key], _ = TranslateJson(config_node)
			}
		case []interface{}:
			if len(elem) == 0 {
				continue
			}

			switch elem[0].(type) {
			case string:
				var list []string
				for j := 0; j < len(elem); j++ {
					text := elem[j].(string)
					res, err := Translate(text, config)

					if err != nil {
						log.Println(err.Error())
						list = append(list, text)
					} else {
						list = append(list, res.Translated_text)
					}
				}
				config.Translated_file[key] = list
				config.State <- models.State{
					Counter: 1,
				}
			case map[string]interface{}:
				var list []map[string]interface{}
				for j := 0; j < len(elem); j++ {
					item := elem[j].(map[string]interface{})

					config_node := config
					config_node.Source_data = item
					config_node.Translated_file = map[string]interface{}{}
					res, err := TranslateJson(config_node)

					if err != nil {
						log.Println(err.Error())
						list = append(list, item)
					} else {
						list = append(list, res)
					}
				}
				config.Translated_file[key] = list
				config.State <- models.State{
					Counter: 1,
				}
			default:
				fmt.Println("Unsupported type : ", reflect.TypeOf(elem[0]), "for key : ", key)
				config.State <- models.State{
					Counter: 1,
				}
			}
		//If we reach a string, we translate it using deepL
		case string:
			res, err := Translate(elem, config)
			if err != nil {
				log.Println(err.Error())
				config.Translated_file[key] = elem
			} else {
				config.Translated_file[key] = res.Translated_text
			}
			config.State <- models.State{
				Counter: 1,
			}
			//If we reach a float or a boolean, we can't really translate it (duh), we simply keep it
		case float64:
			config.Translated_file[key] = elem
			config.State <- models.State{
				Counter: 1,
			}
		case bool:
			config.Translated_file[key] = elem
			config.State <- models.State{
				Counter: 1,
			}
			//Maybe I could remove the float64 and boolean case, and just store elem in default case, but
			//since I don't handle json arrays yet, I prefer not to leave any chance for this program
			//to ungracefully exit
		default:
			fmt.Println("Unsupported type : ", reflect.TypeOf(elem), "for key : ", key)
			config.State <- models.State{
				Counter: 1,
			}
		}
	}
	//The json file has been parsed and translated
	return config.Translated_file, nil
}
