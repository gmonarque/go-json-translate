package models

import "gorm.io/gorm"

type Translation struct {
	gorm.Model
	SourceText     string `gorm:"type:text;index:idx_source_target,priority:1"`
	SourceLang     string `gorm:"type:varchar(10)"`
	TargetLang     string `gorm:"type:varchar(10);index:idx_source_target,priority:2"`
	TranslatedText string `gorm:"type:text"`
}

type TranslationRequest struct {
	Text               []string `json:"text"`
	SourceLang         string   `json:"source_lang,omitempty"`
	TargetLang         string   `json:"target_lang"`
	SplitSentences     string   `json:"split_sentences,omitempty"`
	PreserveFormatting bool     `json:"preserve_formatting,omitempty"`
	Formality          string   `json:"formality,omitempty"`
	GlossaryId         string   `json:"glossary_id,omitempty"`
	TagHandling        string   `json:"tag_handling,omitempty"`
	OutlineDetection   bool     `json:"outline_detection,omitempty"`
}

type TranslationResponse struct {
	DetectedSourceLanguage string `json:"detected_source_language"`
	Text                   string `json:"text"`
}

type Response struct {
	Translations []TranslationResponse `json:"translations"`
}

type State struct {
	Counter int
}

type Config struct {
	SourceData     map[string]interface{}
	TranslatedFile map[string]interface{}
	IgnoredFields  []string
	SourceLang     string
	TargetLang     string
	SourceFilePath string
	APIEndpoint    string
	APIKey         string
	State          chan State
	DB             *gorm.DB
}
