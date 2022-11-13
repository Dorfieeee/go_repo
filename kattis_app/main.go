package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

type markdown struct {
	prefix  string
	postfix string
}

func main() {
	args := os.Args[1:]

	HTMLElements := map[string]markdown{
		"p":      {prefix: "", postfix: ""},
		"center": {prefix: "```\n", postfix: "\n```"},
		"code":   {prefix: "```\n", postfix: "\n```"},
		"h2":     {prefix: "## ", postfix: ""},
		"h3":     {prefix: "### ", postfix: ""},
	}

	var kattisName string
	var destination string
	var kattisURL string
	testDir := "tests"

	usage := "\n\nUsage: kattis [OPTION]... [PROBLEM_NAME] [DESTINATION_PATH]\n" +
		"eg. open.kattis.com/problems/ferryloading3 => kattis ferryloading3\n" +
		"DESTINATION_PATH is [./PROBLEM_NAME/] by default\n" +
		"\nFor help type kattis --help"

	// Check for required [problem_name] argument
	if len(args) > 0 {
		arg := args[0]
		// check for flag
		if strings.HasPrefix(arg, "-") {
			if arg == "--help" {
				fmt.Println(usage)
				fmt.Println("\n\t-t, --test")
				fmt.Println("\t\t test given Python file against all tests")
				os.Exit(0)
			} else if arg == "-t" || arg == "--test" {
				fmt.Println("Running tests...")
				if len(args) > 1 {
					lang := ""
					scriptPath := args[1]
					fileExtension := filepath.Ext(scriptPath)
					// TODO
					// Add support for more languages
					switch fileExtension {
					case ".py":
						lang = "python3"
					default:
						log.Fatal("Testing is not supported for ." + fileExtension + " files")
					}
					// Get lang compiler/interpreter path
					env, err := exec.Command("which", lang).Output()
					if err != nil {
						log.Fatal(lang + " could not be found.")
					}
					// Get all files .in ./tests/ directory
					pwd, err := os.Getwd()
					if err != nil {
						log.Fatal("Could not get current working directory.")
					}
					files, err := os.ReadDir(filepath.Join(pwd, testDir))
					if err != nil {
						log.Fatal("Could not enter tests/ directory")
					}
					for _, file := range files {
						if filepath.Ext(file.Name()) == ".in" {
							cmd := exec.Command(strings.TrimSpace(string(env)), scriptPath)
							in, err := os.Open(filepath.Join(testDir, file.Name()))
							if err != nil {
								log.Fatal("Cound not open " + file.Name())
							}
							cmd.Stdin = in
							result, err := cmd.Output()
							if err != nil {
								log.Fatal("An issue occured during " + scriptPath + " file execution.")
							}
							fmt.Println(string(result))
						}
					}
					// Run test for each of them
					// Run the script with its argument
					// Open its corresponding .out file eg. 0.in => 0.out
					// if .out not present, run it anyway to check for errors at least
					// but state that it could not be tested
					// Loop through both files, comparing each line to each other
					// If any difference, print failure message, continue
					// Print success message

				} else {
					log.Fatal("Missing path to the file to test")
				}
				os.Exit(0)
			}
		} else {
			kattisName = args[0]
			destination = "./" + kattisName
			kattisURL = "https://open.kattis.com/problems/" + kattisName
		}
	} else {
		log.Fatal("Missing the Kattis problem name argument." + usage)
	}

	// Check for optional [destination_path] argument
	if len(args) > 1 {
		destination = args[1]
	}

	// Check whether the destination directory does not already exists
	if filename, _ := os.Stat(destination); filename != nil {
		msg := "Directory " + destination + " already exists.\n" +
			"If you wish to create another directory for the same problem,\n" +
			"provide it as the second argument. [destination_path]" + usage
		log.Fatal(msg)
	}

	c := colly.NewCollector(
		colly.AllowedDomains("open.kattis.com"),
	)

	c.OnHTML("#instructions-container", func(e *colly.HTMLElement) {
		// Create problem's directory
		if err := os.MkdirAll(filepath.Join(destination, "tests"), os.ModePerm); err != nil {
			log.Fatal("Directory " + destination + " already exists.")
		}
		fmt.Println("Parsing HTML...")
		// Create README.md for problem's description
		readmeFile, err := os.Create(destination + "/README.md")
		if err != nil {
			log.Fatal(err)
		}
		defer readmeFile.Close()

		// Write to README.md
		title := e.ChildText(".book-page-heading")
		readmeFile.WriteString("# " + title)
		readmeFile.WriteString("\n[" + kattisURL + "]")

		body := e.DOM.Find(".problembody").First()
		body.Children().Each(func(i int, s *goquery.Selection) {
			nodename := goquery.NodeName(s)
			if md, ok := HTMLElements[nodename]; ok {
				readmeFile.WriteString("\n\n" + md.prefix + s.Text() + md.postfix)
			}
		})

		// Parse test samples and create .in and .out files for each
		body.Find(".sample tbody").Each(func(i int, s *goquery.Selection) {
			titles := s.Find("tr").First().Children()
			inout := s.Find("tr").Last().Children()

			// Input
			inContent := strings.TrimSpace(inout.First().Text()) + "\n"
			readmeFile.WriteString("\n\n### " + titles.First().Text())
			readmeFile.WriteString("\n" + HTMLElements["center"].prefix + inContent + HTMLElements["center"].postfix)
			if inFile, err := os.Create(filepath.Join(destination, "tests", fmt.Sprint(i)+".in")); err != nil {
				fmt.Println(err.Error())
			} else {
				inFile.WriteString(inContent)
			}

			// Output
			outContent := strings.TrimSpace(inout.Last().Text()) + "\n"
			readmeFile.WriteString("\n### " + titles.Last().Text())
			readmeFile.WriteString("\n" + HTMLElements["center"].prefix + outContent + HTMLElements["center"].postfix)
			if outFile, err := os.Create(filepath.Join(destination, "tests", fmt.Sprint(i)+".out")); err != nil {
				fmt.Println(err.Error())
			} else {
				outFile.WriteString(outContent)
			}
		})

		// Create python file
		if pyFile, err := os.Create(filepath.Join(destination, kattisName+".py")); err != nil {
			fmt.Println(err.Error())
		} else {
			pyFile.WriteString("# Measure once, cut twice! ~ Sun Tzu\n")
		}
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Searching", r.URL.String())
	})

	c.OnError(func(r *colly.Response, err error) {
		if err.Error() == "Not Found" {
			//
			log.Fatal("Problem '" + kattisName + "' was not found.")
		}
	})

	c.OnScraped(func(r *colly.Response) {
		fmt.Println("Problem structure created successfully!")
	})

	c.Visit(kattisURL)
}
