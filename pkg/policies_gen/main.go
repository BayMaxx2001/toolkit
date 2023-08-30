package main

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"unicode"

	"gopkg.in/yaml.v2"
)

type resourceConfig map[string][]string
type groupsActionsConfig map[string][]string

type policiesConfig struct {
	FileName       string              `yaml:"name"`
	Output         string              `yaml:"output"`
	Template       string              `yaml:"template"`
	Package        string              `yaml:"package"`
	Groups         groupsActionsConfig `yaml:"groups"`
	ResourceConfig resourceConfig      `yaml:"resource"`
}

type namePolicies struct {
	Private string
	Public  string
	Type    string
}

type templateDataGenerate struct {
	Groups    map[namePolicies][]namePolicies
	Resources map[namePolicies][]namePolicies
	Package   string
}

//go:embed template_policies.tmpl
var templatePolicies embed.FS

func main() {
	if len(os.Args) != 2 {
		log.Fatal(errors.New("invalid arguments"))
	}

	var config policiesConfig
	filepathConfig, err := filepath.Abs(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	fileConfig, err := os.ReadFile(filepathConfig)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(fileConfig, &config)
	if err != nil {
		log.Fatal(err)
	}

	filepathOutput := filepath.Join(filepath.Dir(filepathConfig), fmt.Sprintf("%s/%s.go", config.Output, config.FileName))
	f, err := os.Create(filepathOutput)
	if err != nil {
		log.Fatal(err)
	}

	defer func() { _ = f.Close() }()

	var buffer bytes.Buffer
	data := convertConfigToTemplateData(config)
	tmpl := template.Must(template.New("template_policies.tmpl").ParseFS(templatePolicies, "*.tmpl"))

	err = tmpl.Execute(&buffer, data)
	if err != nil {
		log.Fatal(err)
	}

	bytesFormat, err := format.Source(buffer.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	_, err = f.Write(bytesFormat)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("generate successfully")
}

var link = regexp.MustCompile("(^[A-Za-z])|_([A-Za-z])")

func snakeToCamelCase(str string) string {
	return link.ReplaceAllStringFunc(str, func(s string) string {
		return strings.ToUpper(strings.Replace(s, "_", "", -1))
	})
}

func dashToCamelCase(str string) string {
	str = strings.ReplaceAll(str, "-", "_")
	return snakeToCamelCase(str)
}

func capitalize(s string) string {
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func unCapitalize(s string) string {
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func convertConfigToTemplateData(cfg policiesConfig) templateDataGenerate {
	var templateData templateDataGenerate
	templateData.Groups = make(map[namePolicies][]namePolicies)
	templateData.Resources = make(map[namePolicies][]namePolicies)
	templateData.Package = cfg.Package

	// group actions
	for key, actions := range cfg.Groups {
		var nameActions []namePolicies
		for _, action := range actions {
			kn := namePolicies{
				Private: unCapitalize(snakeToCamelCase(action)),
				Public:  capitalize(dashToCamelCase(action)),
			}
			nameActions = append(nameActions, kn)
		}

		keyName := namePolicies{
			Private: unCapitalize(snakeToCamelCase(key)),
			Public:  capitalize(dashToCamelCase(key)),
		}
		templateData.Groups[keyName] = nameActions
	}

	// resource actions
	for key, actions := range cfg.ResourceConfig {
		var nameActions []namePolicies

		for _, action := range actions {
			kn := namePolicies{
				Private: unCapitalize(snakeToCamelCase(action)),
				Public:  capitalize(dashToCamelCase(action)),
			}
			if !isActionInGroup(action, cfg.Groups) {
				kn.Type = "string"
			}
			nameActions = append(nameActions, kn)
		}

		keyName := namePolicies{
			Private: unCapitalize(snakeToCamelCase(key)),
			Public:  capitalize(dashToCamelCase(key)),
		}

		templateData.Resources[keyName] = nameActions
	}

	return templateData
}

func isActionInGroup(str string, group groupsActionsConfig) bool {
	for key, _ := range group {
		if capitalize(dashToCamelCase(str)) == capitalize(dashToCamelCase(key)) {
			return true
		}
	}
	return false
}
