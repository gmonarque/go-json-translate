package models

import "gorm.io/gorm"

type Translation struct {
	Source_text     string `gorm:"string"`
	Source_lang     string `gorm:"string"`
	Target_lang     string `gorm:"string"`
	Translated_text string `gorm:"string"`
}

type TranslationRequest struct {
	Auth_key            string `json:"auth_key"`              //required
	Text                string `json:"text"`                  //required
	Source_lang         string `json:"source_lang,omitempty"` //optional
	Target_lang         string `json:"target_lang"`           //required
	Split_sentences     string `json:"split_sentences"`       //optional
	Preserve_formatting string `json:"preserve_formatting"`   //optional
	Formality           string `json:"formality"`             //optional
	Glossary_id         string `json:"glossary_id"`           //optional
}

type TranslationResponse struct {
	Detected_source_language string `json:"detected_source_language"`
	Text                     string `json:"text"`
}

type Response struct {
	Translations []TranslationResponse `json:"translations"`
}

type State struct {
	Counter int
}

type Config struct {
	Source_data      map[string]interface{}
	Translated_file  map[string]interface{}
	Ignored_fields   []string
	Source_lang      string
	Target_lang      string
	Source_file_path string
	Api_endpoint     string
	Api_key          string
	State            chan State
	DB               *gorm.DB
}
