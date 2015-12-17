package bettermail

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
)

type Template struct {
	*template.Template
}

var styles map[string]template.CSS

func loadTemplates() (templates map[string]*Template) {
	styles = loadStyles()
	funcMap := template.FuncMap{
		"html": func(value interface{}) template.HTML {
			return template.HTML(fmt.Sprint(value))
		},
		"style": func(names ...string) (result template.CSS) {
			for _, name := range names {
				result += styles[name]
			}
			return
		},
	}
	sharedFileNames, err := filepath.Glob("templates/shared/*.html")
	if err != nil {
		log.Panicf("Could not read shared template file names %s", err.Error())
	}
	templateFileNames, err := filepath.Glob("templates/*.html")
	if err != nil {
		log.Panicf("Could not read template file names %s", err.Error())
	}
	templates = make(map[string]*Template)
	for _, templateFileName := range templateFileNames {
		templateName := filepath.Base(templateFileName)
		templateName = strings.TrimSuffix(templateName, filepath.Ext(templateName))
		fileNames := make([]string, 0, len(sharedFileNames)+2)
		fileNames = append(fileNames, templateFileName)
		fileNames = append(fileNames, sharedFileNames...)
		_, templateFileName = filepath.Split(fileNames[0])
		parsedTemplate, err := template.New(templateFileName).Funcs(funcMap).ParseFiles(fileNames...)
		if err != nil {
			log.Printf("Could not parse template files for %s: %s", templateFileName, err.Error())
		}
		templates[templateName] = &Template{parsedTemplate}
	}
	return templates
}

func loadStyles() (result map[string]template.CSS) {
	stylesBytes, err := ioutil.ReadFile("config/styles.json")
	if err != nil {
		log.Panicf("Could not read styles JSON: %s", err.Error())
	}
	var stylesJson interface{}
	err = json.Unmarshal(stylesBytes, &stylesJson)
	result = make(map[string]template.CSS)
	if err != nil {
		log.Printf("Could not parse styles JSON %s: %s", stylesBytes, err.Error())
		return
	}
	var parse func(string, map[string]interface{}, *string)
	parse = func(path string, stylesJson map[string]interface{}, currentStyle *string) {
		if path != "" {
			path += "."
		}
		for k, v := range stylesJson {
			switch v.(type) {
			case string:
				*currentStyle += k + ":" + v.(string) + ";"
			case map[string]interface{}:
				nestedStyle := ""
				parse(path+k, v.(map[string]interface{}), &nestedStyle)
				result[path+k] = template.CSS(nestedStyle)
			default:
				log.Printf("Unexpected type for %s in styles JSON, ignoring", k)
			}
		}
	}
	parse("", stylesJson.(map[string]interface{}), nil)
	return
}

func getStyle(name string) string {
	return string(styles[name])
}
