package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

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
	var source_file string
	flag.StringVar(&source_lang, "source_lang", "autodetect", "Current language of the file. Use \"autodetect\" to let deepL guess the language.")
	flag.StringVar(&target_lang, "target_lang", "", "Language the file will be translated in")
	flag.StringVar(&source_file, "source_file", "", "Path of the source .json file")
	flag.Parse()

	if source_lang == "" || target_lang == "" || source_file == "" {
		fmt.Println("Usage example: go run main.go -source_file=file.json -source_lang=fr -target_lang=en")
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

	//Config
	config := models.Config{
		Source_data:      map[string]interface{}{},
		Translated_file:  map[string]interface{}{},
		Source_lang:      source_lang,
		Target_lang:      target_lang,
		Source_file_path: source_file,
		Api_endpoint:     ini.Section("").Key("DEEPL_API_ENDPOINT").String(),
		Api_key:          ini.Section("").Key("DEEPL_API_KEY").String(),
		State:            make(chan models.State),
		DB:               DB,
	}

	//Decoding source json file to something we can work with
	directory, filename := path.Split(config.Source_file_path)

	config.Source_data, err = translator.ReadJson(config.Source_file_path)
	if err != nil {
		log.Fatal(err)
	}

	//Initializing the map we are going to use to store the translated data
	config.Translated_file = make(map[string]interface{})

	//Translating the source file
	done := make(chan bool)

	go func() {
		config.Translated_file, err = translator.TranslateJson(config)

		if err != nil {
			log.Fatal(err)
		}

		done <- true
	}()

	//Progress bar thinghy
	bar := progressbar.NewOptions(-1,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSpinnerType(69),
		progressbar.OptionSetWidth(15),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetDescription("[cyan]Translating file..."),
	)

	running := true

	for running {
		select {
		case state := <-config.State:
			bar.Add(state.Counter)
		case _ = <-done:
			fmt.Println("\nTranslation complete!")
			running = false
			bar.Reset()
			break
		}
	}

	//Encoding the map back to json
	json, err := json.Marshal(config.Translated_file)

	if err != nil {
		log.Fatal(err)
	}

	//Saving the translated json file to the same location as the
	//source file and inserting "translated" to the file name
	err = ioutil.WriteFile(directory+"translated_"+filename, json, 0644)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Translated file is at:", directory+"translated_"+filename)

	os.Exit(0)
}
