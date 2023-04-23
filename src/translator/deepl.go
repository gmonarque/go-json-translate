package translator

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/gmonarque/deepl-json/models"
	"github.com/google/go-querystring/query"
)

// If your json file contains variables between delimiters, they certainly don't need to be translated
// You can define below the opening and closing delimiters, and the text between those won't be translated
var delimiters = [][]string{
	{"{", "}"},
	{"#{", "}"},
	{"[", "]"},
	{"<", ">"},
	{"<", "/>"},
}

func Translate(source_text string, config models.Config) (models.Translation, error) {
	translation := models.Translation{
		Source_text:     source_text,
		Source_lang:     config.Source_lang,
		Target_lang:     config.Target_lang,
		Translated_text: "",
	}

	var count int64

	config.DB.Model(&models.Translation{}).Where("Source_text = ?", source_text).Where("Target_lang = ?", config.Target_lang).Count(&count)

	if count > 0 {
		//Translation already exists, using it
		config.DB.Model(&models.Translation{}).Where("Source_text = ?", source_text).Where("Target_lang = ?", config.Target_lang).First(&translation)
		return translation, nil
	}

	var variables_pre []string = make([]string, 0)

	for _, delimiter := range delimiters {
		r, err := regexp.Compile("(\\" + delimiter[0] + "+)(.+?)(\\" + delimiter[1] + "+)")
		if err != nil {
			log.Println("Incorrect delimiters:", delimiter[0], " ", delimiter[1])
			return translation, err
		}

		matches := r.FindAllString(source_text, -1)
		variables_pre = append(variables_pre, matches...)
	}

	request := models.TranslationRequest{
		Auth_key:            config.Api_key,
		Text:                source_text,
		Target_lang:         config.Target_lang,
		Split_sentences:     "",
		Preserve_formatting: "",
		Formality:           "",
		Glossary_id:         "",
	}

	if !strings.EqualFold(config.Source_lang, "autodetect") {
		request.Source_lang = config.Source_lang
	}

	query, _ := query.Values(request)

	resp, err := http.PostForm(config.Api_endpoint, query)

	if err != nil {
		return translation, err
	}

	if resp.StatusCode != 200 {
		return translation, errors.New(resp.Status)
	}

	var res models.Response
	json.NewDecoder(resp.Body).Decode(&res)

	if len(res.Translations) == 0 {
		return translation, errors.New("Response does not contain any translation")
	}

	translation.Translated_text = res.Translations[0].Text

	if len(variables_pre) > 0 {
		var variables_post []string = make([]string, 0)
		for _, delimiter := range delimiters {
			r, err := regexp.Compile("(\\" + delimiter[0] + "+)(.+?)(\\" + delimiter[1] + "+)")

			if err != nil {
				log.Println("Incorrect delimiters:", delimiter[0], " ", delimiter[1])
				return translation, err
			}

			matches := r.FindAllString(translation.Translated_text, -1)
			variables_post = append(variables_post, matches...)
		}

		if len(variables_post) == len(variables_pre) {
			for i := 0; i < len(variables_post); i++ {
				translation.Translated_text = strings.Replace(translation.Translated_text, variables_post[i], variables_pre[i], 1)
			}
		}
	}

	config.DB.Create(&translation)

	return translation, nil
}
