package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Default map[string]interface{} `yaml:"default"`
	Output  []interface{}          `yaml:"output"`
	Input   []struct {
		compiledRegex *regexp.Regexp
		Tag           string   `yaml:"tag"`
		Regex         string   `yaml:"regex"`
		Map           []string `yaml:"map"`
	} `yaml:"input"`
}

var config = Config{}
var input = [][]map[string]string{}
var output = []string{}
var skippedOutput = []string{}

func main() {
	log.SetFlags(0)

	log.Println("Starting", os.Args, "...")

	log.Println("Loading config...")

	configBytes, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	log.Println("Configured", config)

	if len(os.Args) < 2 {
		log.Println("ERROR: No files for parsing specified")
		log.Println("\n!!! Please, drag source text files on top of this .exe file icon to convert !!!")
		log.Println("Press any key to exit...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		return
	}

	input = make([][]map[string]string, len(config.Input))
	log.Println("Found", len(config.Input), "regex patterns to scan")

	for _, path := range os.Args[1:] {
		log.Println("\nParsing input file", path)
		parseIn(path)
	}

	log.Println("\nParsing complete")

	for id, inputConfig := range config.Input {
		log.Println("Found", len(input[id]), "items matching regexp", inputConfig.Regex)
	}

	limit := 0
	for _, items := range input {
		if limit == 0 || limit > len(items) {
			limit = len(items)
		}
	}
	log.Println("\nComposing", limit, "output items...")
	regLimit := len(input)
	for id := 0; id < limit; id++ {
		item := map[string]string{}
		for i := 0; i < regLimit; i++ {
			for key, value := range input[i][id] {
				item[key] = value
			}
		}
		// We've got an item, lets format it the right way out
		rows := []string{}
		for _, tag := range config.Output {
			name := tag.(string)
			skipName := false
			if len(name) > 0 && name[0] == "^"[0] {
				name = name[1:]
				skipName = true
			}
			value, ok := item[name]
			if !ok {
				if defaultValue, ok := config.Default[name]; ok {
					value = defaultValue.(string)
				}
			}
			if skipName {
				rows = append(rows, value)
			} else {
				rows = append(rows, name+" = "+value)
			}
		}
		// Format the output line
		output = append(output, "[ "+strings.Join(rows, "\n")+"\n]")
		// Remove used entries from input arrays
		for i := 0; i < regLimit; i++ {
			input[i][id] = nil
		}
	}

	fileNameSuffix := strings.Replace(time.Now().Format(time.RFC3339), ":", ".", -1)

	err = ioutil.WriteFile("./Result_"+fileNameSuffix+".txt", []byte(strings.Join(output, "\n\n")), 0644)
	if err != nil {
		log.Panic(err)
	}

	// Output not used strings
	output = []string{}
	for _, regs := range input {
		for _, items := range regs {
			if source, ok := items["^"]; ok {
				output = append(output, source)
			}
		}
	}
	if len(output) > 0 {
		err = ioutil.WriteFile("./Unused_"+fileNameSuffix+".txt", []byte(strings.Join(output, "\n")), 0644)
		if err != nil {
			log.Panic(err)
		}
	}
}

func parseIn(path string) {
	file, err := os.Open(path)
	if err != nil {
		defer file.Close()

		log.Println("WARNING:", err)
		return
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Compare against each regex
		for id, inputConfig := range config.Input {
			text := strings.TrimSpace(scanner.Text())
			if len(text) == 0 {
				// Ignore empty rows
				break
			}

			if inputConfig.compiledRegex == nil {
				log.Println("\nCompiling regex", id, inputConfig.Regex)
				rg, err := regexp.Compile(inputConfig.Regex)
				if err != nil || rg == nil {
					log.Panic("ERROR: Regex is not valid")
				}
				inputConfig.compiledRegex = rg
				config.Input[id].compiledRegex = rg
				if inputConfig.compiledRegex == nil {
					log.Panic("ERROR: Can't set regex", id)
				}
			}
			res := inputConfig.compiledRegex.FindAllStringSubmatch(text, -1)

			if len(res) > 0 {
				log.Println("\n- matched", text)
				log.Println("- to     ", inputConfig.Regex)
				// Add result to corresponding input array
				item := map[string]string{}
				item[inputConfig.Tag] = res[0][0]
				item["^"] = res[0][0]
				for coord, tag := range inputConfig.Map {
					item[tag] = res[0][1+coord]
				}
				log.Println("- as     ", item)

				input[id] = append(input[id], item)
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
