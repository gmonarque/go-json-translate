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

func ReadJSON(sourceFile string) (map[string]interface{}, error) {
	jsonFile, err := os.Open(sourceFile)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(byteValue, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func TranslateJSON(config models.Config) (map[string]interface{}, error) {
	keys := reflect.ValueOf(config.SourceData).MapKeys()

	for _, key := range keys {
		keyStr := key.Interface().(string)

		if isIgnored(keyStr, config.IgnoredFields) {
			config.TranslatedFile[keyStr] = config.SourceData[keyStr]
			continue
		}

		elem := config.SourceData[keyStr]
		translatedElem, err := translateElement(elem, config)
		if err != nil {
			log.Printf("Error translating key %s: %v", keyStr, err)
			config.TranslatedFile[keyStr] = elem
		} else {
			config.TranslatedFile[keyStr] = translatedElem
		}

		config.State <- models.State{Counter: 1}
	}

	return config.TranslatedFile, nil
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
	case map[string]interface{}:
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

func translateNestedJSON(data map[string]interface{}, config models.Config) (map[string]interface{}, error) {
	configNode := config
	configNode.SourceData = data
	configNode.TranslatedFile = make(map[string]interface{})
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

func translateString(text string, config models.Config) (string, error) {
	res, err := Translate(text, config)
	if err != nil {
		return text, err
	}
	return res.TranslatedText, nil
}
