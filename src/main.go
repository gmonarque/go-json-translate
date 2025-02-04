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

	"github.com/gmonarque/go-json-translate/db"
	"github.com/gmonarque/go-json-translate/models"
	"github.com/gmonarque/go-json-translate/translator"
	"github.com/iancoleman/orderedmap"
	"github.com/schollz/progressbar/v3"
)

func main() {
	// Checking CLI parameters
	sourceLang := flag.String("source_lang", "autodetect", "Current language of the file. Use \"autodetect\" to let DeepL guess the language.")
	targetLang := flag.String("target_lang", "", "Language the file will be translated to")
	sourcePath := flag.String("source_path", "", "Path of the source file(s)")
	outputPath := flag.String("output_path", "", "Path for the output file(s). If not set, output will be in the same folder as the input")
	ignoredFields := flag.String("ignored_fields", "", "Ignored fields separated by semicolon")
	populateDB := flag.String("populate_db", "", "Path to existing translation file to populate the database")
	flag.Parse()

	if *sourceLang == "" || *targetLang == "" || *sourcePath == "" {
		fmt.Println("Usage example: go run main.go -source_path=folder/*.json -output_path=output/ -source_lang=fr -target_lang=en")
		fmt.Println("Available source languages:")
		fmt.Println("BG, CS, DA, DE, EL, EN, ES, ET, FI, FR, HU, ID, IT, JA, KO, LT, LV, NB, NL, PL, PT, RO, RU, SK, SL, SV, TR, UK, ZH")
		fmt.Println("Available target languages:")
		fmt.Println("AR, BG, CS, DA, DE, EL, EN-GB, EN-US, ES, ET, FI, FR, HU, ID, IT, JA, KO, LT, LV, NB, NL, PL, PT-BR, PT-PT, RO, RU, SK, SL, SV, TR, UK, ZH, ZH-HANS, ZH-HANT")
		fmt.Println("Note: Not all source languages can be used as target languages.")
		fmt.Println("For more information, visit: github.com/gmonarque/go-json-translate")
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
	bar := progressbar.NewOptions(-1,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetDescription("[cyan]Translating..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	// Create a channel to receive progress updates
	progressChan := make(chan int)

	// Populate database if requested
	if *populateDB != "" {
		config := models.Config{
			SourceLang: *sourceLang,
			TargetLang: *targetLang,
			DB:         db,
		}
		if err := translator.PopulateDatabase(*populateDB, config); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Database populated successfully.")
		return
	}

	// Translating each JSON file in the source folder
	for _, file := range files {
		_, filename := filepath.Split(file)

		config := models.Config{
			SourceData:     orderedmap.New(),
			TranslatedFile: orderedmap.New(),
			IgnoredFields:  strings.Split(*ignoredFields, ";"),
			SourceLang:     *sourceLang,
			TargetLang:     *targetLang,
			SourceFilePath: file,
			APIEndpoint:    apiEndpoint,
			APIKey:         apiKey,
			ProgressChan:   progressChan,
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

		// Update progress bar
		for {
			select {
			case <-progressChan:
				_ = bar.Add(1)
			case <-done:
				_ = bar.Finish()
				fmt.Printf("\nTranslation of file %s complete!\n", filename)
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

		// Determine the output directory
		var outputDir string
		if *outputPath == "" {
			outputDir = filepath.Dir(*sourcePath)
		} else {
			outputDir = *outputPath
			if !filepath.IsAbs(outputDir) {
				outputDir = filepath.Join(filepath.Dir(*sourcePath), outputDir)
			}

			// Check if the output directory exists
			if _, err := os.Stat(outputDir); os.IsNotExist(err) {
				fmt.Printf("Output directory %s does not exist. Do you want to create it? (y/n): ", outputDir)
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) == "y" {
					if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
						log.Fatalf("Failed to create output directory: %v", err)
					}
					fmt.Printf("Created output directory: %s\n", outputDir)
				} else {
					log.Fatal("Output directory does not exist. Exiting.")
				}
			}
		}

		// Generate the output file path
		outputFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + "_translated_" + *targetLang + filepath.Ext(filename)
		outputFilePath := filepath.Join(outputDir, outputFilename)

		// Saving the translated JSON file to the output location
		if err := os.WriteFile(outputFilePath, buf.Bytes(), 0644); err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Translated file is at: %s\n", outputFilePath)
	}
}
