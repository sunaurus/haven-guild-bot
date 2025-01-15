package main

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
)

type Config struct {
	HavenAPIBaseURL string `json:"haven_api_base_url"`
	HavenAPIToken   string `json:"haven_api_token"`
	DiscordBotToken string `json:"discord_bot_token"`
}

func loadConfig() (*Config, error) {
	file, err := os.Open("config.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &Config{}
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	err = validateConfigStruct(config)
	if err != nil {
		panic(err)
	}

	return config, nil
}

func validateConfigStruct(s interface{}) error {
	v := reflect.ValueOf(s).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				return errors.New("missing required field: " + fieldType.Name)
			}
			field = field.Elem()
		}

		switch field.Kind() {
		case reflect.Struct:
			err := validateConfigStruct(field.Addr().Interface())
			if err != nil {
				return err
			}
		case reflect.String:
			if field.String() == "" {
				return errors.New("missing required field: " + fieldType.Name)
			}
		default:
			panic("unhandled default case")
		}
	}

	return nil
}
