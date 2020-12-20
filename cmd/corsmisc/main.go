package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/drsigned/gos"
	"github.com/logrusorgru/aurora/v3"
)

type options struct {
	all         bool
	concurrency int
	delay       int
	header      string
	method      string
	noColor     bool
	timeout     int
	output      string
	proxy       string
	URLs        string
	verbose     bool
}

type result struct {
	ACAO []string `json:"acao,omitempty"`
	ACAC string   `json:"acac,omitempty"`
}

var (
	o  options
	au aurora.Aurora
)

func banner() {
	fmt.Fprintln(os.Stderr, aurora.BrightBlue(`
                              _
  ___ ___  _ __ ___ _ __ ___ (_)___  ___
 / __/ _ \| '__/ __| '_ `+"`"+` _ \| / __|/ __|
| (_| (_) | |  \__ \ | | | | | \__ \ (__
 \___\___/|_|  |___/_| |_| |_|_|___/\___| v1.1.0
`).Bold())
}

func init() {
	flag.BoolVar(&o.all, "all", false, "")
	flag.IntVar(&o.concurrency, "c", 20, "")
	flag.IntVar(&o.delay, "d", 100, "")
	flag.StringVar(&o.header, "H", "", "")
	flag.StringVar(&o.method, "X", "GET", "")
	flag.StringVar(&o.proxy, "x", "", "")
	flag.BoolVar(&o.noColor, "nc", false, "")
	flag.StringVar(&o.output, "o", "", "")
	flag.IntVar(&o.timeout, "timeout", 10, "")
	flag.StringVar(&o.URLs, "urls", "", "")
	flag.BoolVar(&o.verbose, "v", false, "")

	flag.Usage = func() {
		banner()

		h := "USAGE:\n"
		h += "  corsmisc [OPTIONS]\n"

		h += "\nOPTIONS:\n"
		h += "  -all            test all Origin's\n"
		h += "  -c              concurrency level (default: 50)\n"
		h += "  -d              delay between requests (default: 100ms)\n"
		h += "  -H              header `\"Name: Value\"`, separated by colon\n"
		h += "  -nc             no color mode\n"
		h += "  -o              JSON output file\n"
		h += "  -timeout        HTTP request timeout (default: 10s)\n"
		h += "  -urls           list of urls (use `-` to read stdin)\n"
		h += "  -UA             HTTP user agent\n"
		h += "  -X              HTTP method to use (default: GET)\n"
		h += "  -x              HTTP Proxy URL\n"
		h += "  -v              verbose mode\n"

		fmt.Fprintf(os.Stderr, h)
	}

	flag.Parse()

	au = aurora.NewAurora(!o.noColor)
}

func main() {
	if o.URLs == "" {
		os.Exit(1)
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

	wg := new(sync.WaitGroup)
	rslt := make(map[string]result)
	delay := time.Duration(o.delay) * time.Millisecond

	for i := 0; i < o.concurrency; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			tr := &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   time.Duration(o.timeout) * time.Second,
					KeepAlive: time.Second,
				}).DialContext,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}

			if o.proxy != "" {
				if p, err := url.Parse(o.proxy); err == nil {
					tr.Proxy = http.ProxyURL(p)
				}
			}

			client := &http.Client{
				Transport: tr,
				Timeout:   time.Duration(o.timeout) * time.Second,
			}

		FOR_EVERY_URL:
			for URL := range URLs {
				parsedURL, err := gos.ParseURL(URL)
				if err != nil {
					if o.verbose {
						fmt.Fprintf(os.Stderr, err.Error())
					}

					continue FOR_EVERY_URL
				}

				origins := []string{
					// wildcard (*)
					"*",
					// whitelisted null origin value
					"null",
					// basic origin reflection
					fmt.Sprintf("%s://corsmisc.com", parsedURL.Scheme),
					// another TLD
					fmt.Sprintf("%s://%s.anothertld", parsedURL.Scheme, parsedURL.DomainName),
					// prefix
					fmt.Sprintf("%s://%s.corsmisc.com", parsedURL.Scheme, parsedURL.DomainName),
					fmt.Sprintf("%s://%s.%s.corsmisc.com", parsedURL.Scheme, parsedURL.DomainName, parsedURL.TLD),
					// suffix
					fmt.Sprintf("%s://corsmisc.%s.%s", parsedURL.Scheme, parsedURL.DomainName, parsedURL.TLD),
					fmt.Sprintf("%s://corsmisc.com.%s.%s", parsedURL.Scheme, parsedURL.DomainName, parsedURL.TLD),
					// unescaped dot
					fmt.Sprintf("%s://corsmisc%s.%s", parsedURL.Scheme, parsedURL.DomainName, parsedURL.TLD),
					// third party origins
					"https://whatever.github.io",
					"http://jsbin.com",
					"https://codepen.io",
					"https://jsfiddle.net",
					"http://www.webdevout.net",
					"https://repl.it",
				}

				chars := []string{"_", "-", "+", "$", "{", "}", "^", "%60", "!", "~", "`", ";", "|", "&", "(", ")", "*", "'", "\"", "=", "%0b"}

				for _, char := range chars {
					origins = append(
						origins,
						fmt.Sprintf(
							"%s://%s.%s%s.corsmisc.com",
							parsedURL.Scheme,
							parsedURL.DomainName,
							parsedURL.TLD,
							char,
						),
					)
				}

			FOR_EVERY_ORIGIN:
				for _, origin := range origins {
					if !o.all {
						if len(rslt[URL].ACAO) > 0 {
							continue FOR_EVERY_URL
						}
					}

					time.Sleep(delay)

					req, err := http.NewRequest(o.method, URL, nil)
					if err != nil {
						log.Fatalln(err)
					}

					req.Header.Set("Origin", origin)

					res, err := client.Do(req)
					if err != nil {
						if o.verbose {
							fmt.Fprintf(os.Stderr, err.Error())
						}

						continue FOR_EVERY_ORIGIN
					}

					if res != nil {
						io.Copy(ioutil.Discard, res.Body)
						res.Body.Close()
					}

					acao := res.Header.Get("Access-Control-Allow-Origin")

					if acao == origin {
						ACAO := append(rslt[URL].ACAO, acao)

						rslt[URL] = result{
							ACAO: ACAO,
							ACAC: res.Header.Get("Access-Control-Allow-Credentials"),
						}
					}
				}
			}
		}()
	}

	wg.Wait()

	if o.output != "" {
		if err := saveResults(o.output, rslt); err != nil {
			log.Fatalln(err)
		}
	}
}

func saveResults(outputPath string, output map[string]result) error {
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
