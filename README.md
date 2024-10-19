# go-json-translate

<p align="center">
  <img width="300" alt="go-json-translate" src="images/go-json-translate-logo.svg">
</p>

<p align="center">
  <strong>Translate JSON language files / i18n locales on the spot!</strong>
</p>

<p align="center">
  <a href="https://github.com/gmonarque/go-json-translate/issues">
    <img src="https://img.shields.io/github/issues/gmonarque/go-json-translate" alt="GitHub issues">
  </a>
  <a href="https://github.com/gmonarque/go-json-translate/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/gmonarque/go-json-translate" alt="GitHub license">
  </a>
</p>

## About

go-json-translate is a tool designed to easily translate JSON translation files. It's particularly useful for projects with large language files that need quick and efficient translation.

## Features

- Translate any standard JSON file using DeepL's free API
- Support for JSON files containing strings, numbers, and booleans
- Handles nested JSON files without limits
- Local database for translation caching to improve speed and reduce API calls
- Preserves variables in translated text (e.g., `hello, {name}` -> `bonjour, {name}`)
- Supports various enclosure tags: `{}`, `#{}`, `[]`

## Example

English (Source):
```json
{
  "account": {
    "breadcrumb": "Your Account",
    "nav": {
      "overview": "Overview",
      "orders": "Orders",
      "messages": "Messages ({num_new_messages})"
    }
  }
}
```

French (Translated):
```json
{
  "account": {
    "breadcrumb": "Votre compte",
    "nav": {
      "overview": "Vue d'ensemble",
      "orders": "Commandes",
      "messages": "Messages ({num_new_messages})"
    }
  }
}
```

## Installation

### Prerequisites

- Go 1.17 or higher
- DeepL API key

### Install from Source

```sh
git clone https://github.com/gmonarque/go-json-translate
cd go-json-translate
go mod download
go build
```

## Configuration

Before using go-json-translate, configure the `config.ini` file:

```ini
DEEPL_API_ENDPOINT = https://api-free.deepl.com/v2/translate
DEEPL_API_KEY = <your_api_key>
```

## Usage

```sh
./go-json-translate -source_path=<path> -output_path=<path> -source_lang=<lang> -target_lang=<lang> [-ignored_fields=<fields>]
```

### Options

- `-source_path`: Path of the source JSON file(s)
- `-output_path`: Path for the output file(s)
- `-source_lang`: Current language of the file (use "autodetect" to let DeepL guess)
- `-target_lang`: Language to translate the file into
- `-ignored_fields`: (Optional) Fields to ignore, separated by semicolons

### Example

```sh
./go-json-translate -source_path=folder/*.json -output_path=output/*.json -source_lang=fr -target_lang=en
```

## Available Languages

### Source Languages
BG, CS, DA, DE, EL, EN, ES, ET, FI, FR, HU, ID, IT, JA, KO, LT, LV, NB, NL, PL, PT, RO, RU, SK, SL, SV, TR, UK, ZH

### Target Languages
AR, BG, CS, DA, DE, EL, EN-GB, EN-US, ES, ET, FI, FR, HU, ID, IT, JA, KO, LT, LV, NB, NL, PL, PT-BR, PT-PT, RO, RU, SK, SL, SV, TR, UK, ZH, ZH-HANS, ZH-HANT

Note: Not all source languages can be used as target languages. For the most up-to-date list, refer to [DeepL's API documentation](https://www.deepl.com/docs-api/translating-text/request/).

## Limitations

- Depends on DeepL's free tier API usage limits
- No support for JSON arrays
- Large files may cause instability depending on hardware
- Output JSON order may differ from the source file

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Community Translations

If you've translated something that could be useful to others, please submit a merge request with your translated file in the `community-translated-files` folder.

## License

This project is licensed under the [MIT License](LICENSE).

## Contact

For any questions or issues, please [open an issue](https://github.com/gmonarque/go-json-translate/issues) on GitHub.

---

<p align="center">
  <a href="https://gmsec.fr/">Personal Website</a>
</p>
