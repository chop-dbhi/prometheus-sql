package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-playground/locales/en"

	ut "github.com/go-playground/universal-translator"
	validator "gopkg.in/go-playground/validator.v9"
	en_translations "gopkg.in/go-playground/validator.v9/translations/en"
	yaml "gopkg.in/yaml.v2"
)

var (
	// use a single instance of Validate, it caches struct info
	validate *validator.Validate
	trans    ut.Translator
)

// Default config values
var (
	DefaultHost        = ""
	DefaultTimeout     = time.Minute
	DefaultInterval    = time.Minute * 5
	DefaultService     = ""
	DefaultQueriesFile = "queries.yml"
	DefaultQueriesDir  = ""
	DefaultPort        = 8080
	DefaultConfFile    = ""
)

// Config is the base data structure.
type Config struct {
	Defaults    DefaultsData          `yaml:"defaults" validate:"dive"`
	DataSources map[string]DataSource `yaml:"data-sources" validate:"dive"`
}

// DefaultsData defines the possible default values to define.
type DefaultsData struct {
	DataSourceRef     string        `yaml:"data-source"`
	QueryInterval     time.Duration `yaml:"query-interval" validate:"gt=0"`
	QueryTimeout      time.Duration `yaml:"query-timeout" validate:"gt=0"`
	QueryValueOnError string        `yaml:"query-value-on-error"`
}

// DataSource is configuration a data source which must be supported by sql-agent.
type DataSource struct {
	Driver     string                 `yaml:"driver" validate:"required"`
	Properties map[string]interface{} `yaml:"properties" validate:"required"`
}

// Query defines a SQL statement and parameters as well as configuration for the monitoring behavior
type Query struct {
	Name          string                 `validate:"required"`
	DataSourceRef string                 `yaml:"data-source"`
	Driver        string                 `validate:"required"`
	Connection    map[string]interface{} `validate:"required"`
	SQL           string                 `validate:"required"`
	Params        map[string]interface{}
	Interval      time.Duration `validate:"gt=0"` // default 5m
	Timeout       time.Duration `validate:"gt=0"` // default 1m
	DataField     string        `yaml:"data-field"`
	ValueOnError  string        `yaml:"value-on-error"`
}

// QueryList is a array or Queries
type QueryList []*Query

func createDefaultsData() DefaultsData {
	return DefaultsData{
		DataSourceRef:     "",
		QueryInterval:     DefaultInterval,
		QueryTimeout:      DefaultTimeout,
		QueryValueOnError: "",
	}
}

func newConfig() *Config {
	return &Config{Defaults: createDefaultsData()}
}

func appendDefaults(c Config) Config {
	if c.Defaults.QueryInterval == 0 {
		c.Defaults.QueryInterval = DefaultInterval
	}
	if c.Defaults.QueryTimeout == 0 {
		c.Defaults.QueryTimeout = DefaultTimeout
	}
	return c
}

func getValidator() *validator.Validate {
	if validate == nil {
		en := en.New()
		uni := ut.New(en, en)
		trans, _ := uni.GetTranslator("en")
		validate = validator.New()
		en_translations.RegisterDefaultTranslations(validate, trans)
	}
	return validate
}

func validateStruct(c interface{}) error {
	if err := getValidator().Struct(c); err != nil {
		errs := err.(validator.ValidationErrors)
		for _, e := range errs {
			// We just take the first error
			return fmt.Errorf("Validation error: %s", e.Translate(trans))
		}
	}
	return nil
}

func loadConfig(file string) (*Config, error) {
	log.Printf("Load config from file [%s]", file)
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("Error reading config file: %s", err)
	}
	var c Config
	if err := yaml.Unmarshal([]byte(b), &c); err != nil {
		return nil, fmt.Errorf("Error decoding config file: %s", err)
	}
	c = appendDefaults(c)
	if err := validateStruct(c); err != nil {
		return nil, err
	}
	return &c, err
}

func loadQueryConfig(queriesFile string, config *Config) (QueryList, error) {
	log.Printf("Load queries from file [%s]", queriesFile)
	// Read queries for request body.
	file, err := os.Open(queriesFile)
	if err != nil {
		return nil, fmt.Errorf("Error opening queries file: %s", err)
	}
	defer file.Close()
	return decodeQueries(file, config)
}

func useFallbackIfEmpty(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func useFallbackIfZero(value time.Duration, fallback time.Duration) time.Duration {
	if value == 0 {
		return fallback
	}
	return value
}

func decodeQueries(r io.Reader, config *Config) (QueryList, error) {
	if config == nil {
		return nil, errors.New("Bug! Config must not be nil")
	}

	queries := make(QueryList, 0)
	parsedQueries := make([]map[string]*Query, 0)

	b, err := ioutil.ReadAll(r)

	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(b, &parsedQueries); err != nil {
		return nil, err
	}

	for _, data := range parsedQueries {
		for k, q := range data {
			q.Name = k
			q.DataSourceRef = useFallbackIfEmpty(q.DataSourceRef, config.Defaults.DataSourceRef)
			if q.Driver == "" {
				if q.DataSourceRef != "" && len(config.DataSources) > 0 {
					var ds = config.DataSources[q.DataSourceRef]
					q.Driver = ds.Driver
					q.Connection = ds.Properties
				}
				if q.Driver == "" {
					return nil, fmt.Errorf("No data source or driver is specified for query [%s]", q.Name)
				}
			}
			if q.SQL == "" {
				return nil, errors.New("SQL statement required")
			}
			q.Interval = useFallbackIfZero(q.Interval, config.Defaults.QueryInterval)
			q.Timeout = useFallbackIfZero(q.Timeout, config.Defaults.QueryTimeout)
			if q.ValueOnError == "" && config.Defaults.QueryValueOnError != "" {
				q.ValueOnError = config.Defaults.QueryValueOnError
			}
			q.DataField = strings.ToLower(q.DataField)
			if err := validateStruct(q); err != nil {
				return nil, err
			}

			queries = append(queries, q)
		}
	}
	return queries, nil
}

func loadQueriesInDir(path string, config *Config) (QueryList, error) {
	log.Printf("Load queries from directory [%s]", path)
	queries := make(QueryList, 0)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		fn := f.Name()
		if strings.HasSuffix(fn, ".yml") {
			fn := fmt.Sprintf("%s/%s", strings.TrimRight(path, "/"), fn)
			log.Println("Loading", fn)
			file, err := os.Open(fn)
			if err != nil {
				return nil, err
			}
			q, err := decodeQueries(file, config)
			if err != nil {
				return nil, err
			}
			queries = append(queries, q...)
			file.Close()
		}
	}
	return queries, nil
}
