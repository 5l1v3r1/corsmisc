package main

import (
	"bufio"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/drsigned/gos"
	"github.com/logrusorgru/aurora/v3"
)

type options struct {
	concurrency int
	delay       int
	header      string
	method      string
	noColor     bool
	timeout     int
	URLs        string
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
 \___\___/|_|  |___/_| |_| |_|_|___/\___| v1.0.0
`).Bold())
}

func init() {
	flag.IntVar(&o.concurrency, "c", 20, "")
	flag.StringVar(&o.header, "H", "", "")
	flag.StringVar(&o.method, "X", "GET", "")
	flag.BoolVar(&o.noColor, "nc", false, "")
	flag.IntVar(&o.timeout, "timeout", 10, "")
	flag.StringVar(&o.URLs, "urls", "", "")

	flag.Usage = func() {
		banner()

		h := "USAGE:\n"
		h += "  corsmisc [OPTIONS]\n"

		h += "\nOPTIONS:\n"
		h += "   -c              number of concurrent threads. (default: 50)\n"
		h += "   -delay          delay between requests (ms) (default: 100)\n"
		h += "   -H              Header `\"Name: Value\"`, separated by colon. Multiple -H flags are accepted.\n"
		h += "   -nc             no color mode\n"
		h += "   -timeout        HTTP request timeout in seconds. (default: 10)\n"
		h += "   -urls           list of urls (use `-` to read stdin)\n"
		h += "   -UA             HTTP user agent\n"
		h += "   -X              HTTP method to use (default: GET)\n"

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

	delay := time.Duration(o.delay) * time.Millisecond

	for i := 0; i < o.concurrency; i++ {
		wg.Add(1)

		time.Sleep(delay)

		go func() {
			defer wg.Done()

			tr := &http.Transport{
				MaxIdleConns:    30,
				IdleConnTimeout: time.Second,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				DialContext: (&net.Dialer{
					Timeout:   time.Duration(o.timeout) * time.Second,
					KeepAlive: time.Second,
				}).DialContext,
			}

			re := func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}

			client := &http.Client{
				Transport:     tr,
				CheckRedirect: re,
				Timeout:       time.Second * 10,
			}

			for URL := range URLs {
				parsedURL, err := gos.ParseURL(URL)
				if err != nil {
					log.Fatalln(err)
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

				specialChars := []string{"_", "-", "+", "$", "{", "}", "^", "%60", "!", "~", "`", ";", "|", "&", "(", ")", "*", "'", "\"", "=", "%0b"}

				for _, char := range specialChars {
					origins = append(origins, fmt.Sprintf("%s://%s.%s%s.corsmisc.com", parsedURL.Scheme, parsedURL.DomainName, parsedURL.TLD, char))
				}

				for _, origin := range origins {
					req, err := http.NewRequest(o.method, URL, nil)
					if err != nil {
						return
					}
					req.Header.Set("Origin", origin)

					res, err := client.Do(req)
					if err != nil {
						return
					}

					if res != nil {
						io.Copy(ioutil.Discard, res.Body)
						res.Body.Close()
					}

					acao := res.Header.Get("Access-Control-Allow-Origin")
					acac := res.Header.Get("Access-Control-Allow-Credentials")

					fmt.Println("[ACAO:" + acao + "] [ACAC:" + acac + "] - " + URL)
				}
			}
		}()
	}

	wg.Wait()
}
