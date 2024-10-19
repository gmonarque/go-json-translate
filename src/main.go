package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
	"gorm.io/gorm"

	"github.com/gmonarque/deepl-json/db"
	"github.com/gmonarque/deepl-json/models"
	"github.com/gmonarque/deepl-json/translator"
	"github.com/schollz/progressbar/v3"
)

func main() {
	// Checking CLI parameters
	sourceLang := flag.String("source_lang", "autodetect", "Current language of the file. Use \"autodetect\" to let DeepL guess the language.")
	targetLang := flag.String("target_lang", "", "Language the file will be translated to")
	sourcePath := flag.String("source_path", "", "Path of the source file(s)")
	ignoredFields := flag.String("ignored_fields", "", "Ignored fields separated by semicolon")
	flag.Parse()

	if *sourceLang == "" || *targetLang == "" || *sourcePath == "" {
		fmt.Println("Usage example: go run main.go -source_path=folder/*.json -source_lang=fr -target_lang=en")
		fmt.Println("List of languages available at github.com/gmonarque/go-json-translate")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Loading configuration
	cfg, err := ini.Load("config.ini")
	if err != nil {
		log.Fatal(err)
	}

	apiEndpoint := cfg.Section("").Key("DEEPL_API_ENDPOINT").String()
	apiKey := cfg.Section("").Key("DEEPL_API_KEY").String()
	if apiEndpoint == "" || apiKey == "" {
		log.Fatal(errors.New("missing configuration parameters in config.ini"))
	}

	// Creating and/or migrating the GORM database
	db := db.GetDb()
	if err := db.AutoMigrate(&models.Translation{}); err != nil {
		log.Fatal(err)
	}

	// Getting list of JSON files in the source path
	files, err := filepath.Glob(*sourcePath)
	if err != nil {
		log.Fatal(err)
	}

	// Progress bar
	bar := progressbar.NewOptions(len(files),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSpinnerType(69),
		progressbar.OptionSetWidth(15),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetDescription("[cyan]Translating files..."),
	)

	// Translating each JSON file in the source folder
	for _, file := range files {
		directory, filename := filepath.Split(file)

		config := models.Config{
			SourceData:     make(map[string]interface{}),
			TranslatedFile: make(map[string]interface{}),
			IgnoredFields:  strings.Split(*ignoredFields, ";"),
			SourceLang:     *sourceLang,
			TargetLang:     *targetLang,
			SourceFilePath: file,
			APIEndpoint:    apiEndpoint,
			APIKey:         apiKey,
			State:          make(chan models.State),
			DB:             db,
		}

		// Decoding source JSON file
		config.SourceData, err = translator.ReadJSON(file)
		if err != nil {
			log.Fatal(err)
		}

		// Translating the source file
		done := make(chan bool)
		go func() {
			config.TranslatedFile, err = translator.TranslateJSON(config)
			if err != nil {
				log.Fatal(err)
			}
			done <- true
		}()

		for {
			select {
			case state := <-config.State:
				bar.Add(state.Counter)
			case <-done:
				fmt.Printf("\nTranslation of file %s complete!\n", filename)
				bar.Reset()
				goto translationDone
			}
		}
	translationDone:

		// Encoding the map back to JSON
		buf := new(bytes.Buffer)
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "    ")

		if err := enc.Encode(config.TranslatedFile); err != nil {
			log.Fatal(err)
		}

		// Saving the translated JSON file to the same location as the source file
		translatedFilePath := filepath.Join(directory, filename)
		if err := os.WriteFile(translatedFilePath, buf.Bytes(), 0644); err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Translated file is at: %s\n", translatedFilePath)
	}
}
