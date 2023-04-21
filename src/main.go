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
	//Checking CLI parameters
	var source_lang string
	var target_lang string
	var source_path string
	var ignored_fields string

	flag.StringVar(&source_lang, "source_lang", "autodetect", "Current language of the file. Use \"autodetect\" to let deepL guess the language.")
	flag.StringVar(&target_lang, "target_lang", "", "Language the file will be translated in")
	flag.StringVar(&source_path, "source_path", "", "Path of the source file(s)")
	flag.StringVar(&ignored_fields, "ignored_fields", "", "Ignored fields separated by semicolon")
	flag.Parse()

	if source_lang == "" || target_lang == "" || source_path == "" {
		fmt.Println("Usage example: go run main.go -source_path=folder/*.json -source_lang=fr -target_lang=en")
		fmt.Println("List of languages available at github.com/gmonarque/go-json-translate")
		flag.PrintDefaults()
		os.Exit(1)
	}

	//Loading configuration
	ini, err := ini.Load("config.ini")

	if err != nil {
		log.Fatal(err)
	}

	if ini.Section("").Key("DEEPL_API_ENDPOINT").String() == "" || ini.Section("").Key("DEEPL_API_KEY").String() == "" {
		log.Fatal(errors.New("Missing configuration parameters in config.ini"))
	}

	//Creating and/or migrating the GORM database
	var DB *gorm.DB = db.GetDb()

	DB.AutoMigrate(&models.Translation{})

	//Getting list of JSON files in the source path
	files, err := filepath.Glob(source_path)
	if err != nil {
		log.Fatal(err)
	}

	//Progress bar thinghy
	bar := progressbar.NewOptions(len(files),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSpinnerType(69),
		progressbar.OptionSetWidth(15),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetDescription("[cyan]Translating files..."),
	)

	//Translating each JSON file in the source folder
	for _, file := range files {
		//Deciding target file name
		//Creating file name for translated file
		directory, filename := filepath.Split(file)
		translated_filename := filename

		//Config
		config := models.Config{
			Source_data:      map[string]interface{}{},
			Translated_file:  map[string]interface{}{},
			Ignored_fields:   strings.Split(ignored_fields, ";"),
			Source_lang:      source_lang,
			Target_lang:      target_lang,
			Source_file_path: file,
			Api_endpoint:     ini.Section("").Key("DEEPL_API_ENDPOINT").String(),
			Api_key:          ini.Section("").Key("DEEPL_API_KEY").String(),
			State:            make(chan models.State),
			DB:               DB,
		}

		//Initializing the map we are going to use to store the translated data
		config.Translated_file = make(map[string]interface{})

		//Decoding source json file to something we can work with
		config.Source_data, err = translator.ReadJson(file)
		if err != nil {
			log.Fatal(err)
		}

		//Translating the source file
		done := make(chan bool)

		go func() {
			config.Translated_file, err = translator.TranslateJson(config)

			if err != nil {
				log.Fatal(err)
			}

			done <- true
		}()

		running := true

		for running {
			select {
			case state := <-config.State:
				bar.Add(state.Counter)
			case _ = <-done:
				fmt.Println("\nTranslation of file", filename, "complete!")
				running = false
				bar.Reset()
				break
			}
		}

		//Encoding the map back to json
		buf := new(bytes.Buffer)
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "    ")

		err = enc.Encode(config.Translated_file)

		if err != nil {
			log.Fatal(err)
		}

		//Saving the translated json file to the same location as the source file
		err = os.WriteFile(filepath.Join(directory, translated_filename), buf.Bytes(), 0644)

		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Translated file is at:", filepath.Join(directory, translated_filename))

	}

	os.Exit(0)
}
