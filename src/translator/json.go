package translator

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"

	"github.com/gmonarque/go-json-translate/models"
	"github.com/iancoleman/orderedmap"
)

func ReadJSON(sourceFile string) (*orderedmap.OrderedMap, error) {
	jsonFile, err := os.Open(sourceFile)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	result := orderedmap.New()
	if err := json.Unmarshal(byteValue, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func TranslateJSON(config models.Config) (*orderedmap.OrderedMap, error) {
	translatedFile := orderedmap.New()
	keys := config.SourceData.Keys()

	for _, key := range keys {
		elem, _ := config.SourceData.Get(key)

		if isIgnored(key, config.IgnoredFields) {
			translatedFile.Set(key, elem)
			continue
		}

		translatedElem, err := translateElement(elem, config)
		if err != nil {
			log.Printf("Error translating key %s: %v", key, err)
			translatedFile.Set(key, elem)
		} else {
			translatedFile.Set(key, translatedElem)
		}

		config.State <- models.State{Counter: 1}
	}

	return translatedFile, nil
}

func isIgnored(key string, ignoredFields []string) bool {
	for _, ignoredKey := range ignoredFields {
		if key == ignoredKey {
			return true
		}
	}
	return false
}

func translateElement(elem interface{}, config models.Config) (interface{}, error) {
	switch v := elem.(type) {
	case *orderedmap.OrderedMap:
		return translateNestedJSON(v, config)
	case []interface{}:
		return translateArray(v, config)
	case string:
		return translateString(v, config)
	case float64, bool:
		return v, nil
	default:
		return nil, fmt.Errorf("unsupported type: %v", reflect.TypeOf(elem))
	}
}

func translateNestedJSON(data *orderedmap.OrderedMap, config models.Config) (*orderedmap.OrderedMap, error) {
	configNode := config
	configNode.SourceData = data
	return TranslateJSON(configNode)
}

func translateArray(arr []interface{}, config models.Config) ([]interface{}, error) {
	var translatedArr []interface{}
	for _, item := range arr {
		translatedItem, err := translateElement(item, config)
		if err != nil {
			return nil, err
		}
		translatedArr = append(translatedArr, translatedItem)
	}
	return translatedArr, nil
}

func PopulateDatabase(filePath string, config models.Config) error {
	data, err := ReadJSON(filePath)
	if err != nil {
		return err
	}

	return populateDatabaseRecursive(data, config)
}

func populateDatabaseRecursive(data *orderedmap.OrderedMap, config models.Config) error {
	for _, key := range data.Keys() {
		value, _ := data.Get(key)
		switch v := value.(type) {
		case string:
			translation := models.Translation{
				SourceText:     v,
				SourceLang:     config.SourceLang,
				TargetLang:     config.TargetLang,
				TranslatedText: v,
			}
			if err := config.DB.Create(&translation).Error; err != nil {
				return err
			}
		case *orderedmap.OrderedMap:
			if err := populateDatabaseRecursive(v, config); err != nil {
				return err
			}
		case []interface{}:
			for _, item := range v {
				if nestedMap, ok := item.(*orderedmap.OrderedMap); ok {
					if err := populateDatabaseRecursive(nestedMap, config); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func translateString(text string, config models.Config) (string, error) {
	res, err := Translate(text, config)
	if err != nil {
		return text, err
	}
	return res.TranslatedText, nil
}
