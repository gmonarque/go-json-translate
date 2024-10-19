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

var delimiters = [][]string{
	{"{", "}"},
	{"#{", "}"},
	{"[", "]"},
	{"<", ">"},
	{"<", "/>"},
}

func Translate(sourceText string, config models.Config) (models.Translation, error) {
	translation := models.Translation{
		SourceText:     sourceText,
		SourceLang:     config.SourceLang,
		TargetLang:     config.TargetLang,
		TranslatedText: "",
	}

	// Check if translation already exists in the database
	var count int64
	config.DB.Model(&models.Translation{}).Where("source_text = ? AND target_lang = ?", sourceText, config.TargetLang).Count(&count)

	if count > 0 {
		config.DB.Where("source_text = ? AND target_lang = ?", sourceText, config.TargetLang).First(&translation)
		return translation, nil
	}

	variablesPre := extractVariables(sourceText)

	request := models.TranslationRequest{
		AuthKey:    config.APIKey,
		Text:       sourceText,
		TargetLang: config.TargetLang,
	}

	if !strings.EqualFold(config.SourceLang, "autodetect") {
		request.SourceLang = config.SourceLang
	}

	query, err := query.Values(request)
	if err != nil {
		return translation, err
	}

	resp, err := http.PostForm(config.APIEndpoint, query)
	if err != nil {
		return translation, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return translation, errors.New(resp.Status)
	}

	var res models.Response
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return translation, err
	}

	if len(res.Translations) == 0 {
		return translation, errors.New("response does not contain any translation")
	}

	translation.TranslatedText = res.Translations[0].Text

	if len(variablesPre) > 0 {
		variablesPost := extractVariables(translation.TranslatedText)
		if len(variablesPost) == len(variablesPre) {
			for i, v := range variablesPost {
				translation.TranslatedText = strings.Replace(translation.TranslatedText, v, variablesPre[i], 1)
			}
		}
	}

	config.DB.Create(&translation)

	return translation, nil
}

func extractVariables(text string) []string {
	var variables []string
	for _, delimiter := range delimiters {
		r, err := regexp.Compile("(\\" + delimiter[0] + "+)(.+?)(\\" + delimiter[1] + "+)")
		if err != nil {
			log.Printf("Incorrect delimiters: %s %s", delimiter[0], delimiter[1])
			continue
		}
		matches := r.FindAllString(text, -1)
		variables = append(variables, matches...)
	}
	return variables
}
