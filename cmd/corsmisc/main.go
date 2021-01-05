package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/drsigned/corsmisc/pkg/corsmisc"
	"github.com/drsigned/gos"
	"github.com/logrusorgru/aurora/v3"
)

type options struct {
	concurrency int
	method      string
	noColor     bool
	silent      bool
	output      string
	URLs        string
	verbose     bool
}

var (
	o  options
	au aurora.Aurora
	ro corsmisc.Options
)

func banner() {
	fmt.Fprintln(os.Stderr, aurora.BrightBlue(`
                              _
  ___ ___  _ __ ___ _ __ ___ (_)___  ___
 / __/ _ \| '__/ __| '_ `+"`"+` _ \| / __|/ __|
| (_| (_) | |  \__ \ | | | | | \__ \ (__
 \___\___/|_|  |___/_| |_| |_|_|___/\___| v1.3.0
`).Bold())
}

func init() {
	flag.BoolVar(&ro.All, "all", false, "")
	flag.IntVar(&o.concurrency, "c", 20, "")
	flag.IntVar(&ro.Delay, "delay", 100, "")
	flag.BoolVar(&o.noColor, "nC", false, "")
	flag.StringVar(&o.output, "oJ", "", "")
	flag.BoolVar(&o.silent, "s", false, "")
	flag.IntVar(&ro.Timeout, "timeout", 10, "")
	flag.StringVar(&o.URLs, "iL", "", "")
	flag.BoolVar(&o.verbose, "v", false, "")
	flag.StringVar(&ro.Method, "X", "GET", "")
	flag.StringVar(&ro.HTTPProxy, "http-proxy", "", "")

	flag.Usage = func() {
		banner()

		h := "USAGE:\n"
		h += "  corsmisc [OPTIONS]\n"

		h += "\nOPTIONS:\n"
		h += "  -all            test all Origin's\n"
		h += "  -c              concurrency level (default: 50)\n"
		h += "  -delay          delay between requests (default: 100ms)\n"
		h += "  -nC             no color mode\n"
		h += "  -oJ             JSON output file\n"
		h += "  -s              silent mode\n"
		h += "  -timeout        HTTP request timeout (default: 10s)\n"
		h += "  -iL             list of urls (use `-iL -` to read from stdin)\n"
		h += "  -UA             HTTP user agent\n"
		h += "  -v              verbose mode\n"
		h += "  -X              HTTP method to use (default: GET)\n"
		h += "  -http-proxy     HTTP Proxy URL\n"

		fmt.Fprintf(os.Stderr, h)
	}

	flag.Parse()

	au = aurora.NewAurora(!o.noColor)
}

func main() {
	if o.URLs == "" {
		os.Exit(1)
	}

	if !o.silent {
		banner()
	}

	URLs := make(chan string, o.concurrency)

	go func() {
		defer close(URLs)

		var scanner *bufio.Scanner

		if o.URLs == "-" {
			if !gos.HasStdin() {
				log.Fatalln(errors.New("no stdin"))
			}

			scanner = bufio.NewScanner(os.Stdin)
		} else {
			openedFile, err := os.Open(o.URLs)
			if err != nil {
				log.Fatalln(err)
			}

			defer openedFile.Close()

			scanner = bufio.NewScanner(openedFile)
		}

		for scanner.Scan() {
			URLs <- scanner.Text()
		}

		if scanner.Err() != nil {
			log.Fatalln(scanner.Err())
		}
	}()

	output := []corsmisc.Result{}
	mutex := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	for i := 0; i < o.concurrency; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			runner, err := corsmisc.New(ro)
			if err != nil {
				log.Fatalln(err)
			}

			for URL := range URLs {
				result, err := runner.Run(URL)
				if err != nil {
					if o.verbose {
						fmt.Fprintf(os.Stderr, err.Error()+"\n")
					}

					continue
				}

				if result.ACAC == "true" {
					fmt.Println("[", au.BrightGreen("VULENERABLE").Bold(), "]", URL, "-H", au.BrightBlue("Origin: "+result.ACAO[0]).Italic())

					mutex.Lock()
					output = append(output, result)
					mutex.Unlock()
				} else {
					if !o.silent {
						fmt.Println("[", au.BrightRed("NOT VULENERABLE").Bold(), "]", result.URL)
					}
				}
			}
		}()
	}

	wg.Wait()

	if o.output != "" {
		if err := saveResults(o.output, output); err != nil {
			log.Fatalln(err)
		}
	}
}

func saveResults(outputPath string, output []corsmisc.Result) error {
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		directory, filename := path.Split(outputPath)

		if _, err := os.Stat(directory); os.IsNotExist(err) {
			if directory != "" {
				err = os.MkdirAll(directory, os.ModePerm)
				if err != nil {
					return err
				}
			}
		}

		if strings.ToLower(path.Ext(filename)) != ".json" {
			outputPath = outputPath + ".json"
		}
	}

	outputJSON, err := json.MarshalIndent(output, "", "\t")
	if err != nil {
		return err
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}

	defer outputFile.Close()

	_, err = outputFile.WriteString(string(outputJSON))
	if err != nil {
		return err
	}

	return nil
}
