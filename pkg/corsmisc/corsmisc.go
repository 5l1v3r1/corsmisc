package corsmisc

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/drsigned/gos"
)

type Options struct {
	All       bool
	Delay     int
	HTTPProxy string
	Method    string
	Timeout   int
}

type Corsmisc struct {
	Client  *http.Client
	Options Options
}

type Result struct {
	URL  string   `json:"url,omitempty"`
	ACAO []string `json:"acao,omitempty"`
	ACAC string   `json:"acac,omitempty"`
}

func New(options Options) (Corsmisc, error) {
	corsmisc := Corsmisc{}

	tr := &http.Transport{
		MaxIdleConns:    30,
		IdleConnTimeout: time.Second,
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(options.Timeout) * time.Second,
			KeepAlive: time.Second,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	if options.HTTPProxy != "" {
		if proxyURL, err := url.Parse(options.HTTPProxy); err == nil {
			tr.Proxy = http.ProxyURL(proxyURL)
		}
	}

	re := func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	corsmisc.Client = &http.Client{
		Timeout:       time.Duration(options.Timeout) * time.Second,
		Transport:     tr,
		CheckRedirect: re,
	}

	return corsmisc, nil
}

func (corsmisc Corsmisc) Run(URL string) (Result, error) {
	var result Result

	parsedURL, err := gos.ParseURL(URL)
	if err != nil {
		return result, err
	}

	result.URL = parsedURL.String()

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

	// special characters
	chars := []string{"_", "-", "+", "$", "{", "}", "^", "%60", "!", "~", "`", ";", "|", "&", "(", ")", "*", "'", "\"", "=", "%0b"}

	for _, char := range chars {
		origins = append(
			origins,
			fmt.Sprintf("%s://%s.%s%s.corsmisc.com", parsedURL.Scheme, parsedURL.DomainName, parsedURL.TLD, char),
		)
	}

	for _, origin := range origins {
		if !corsmisc.Options.All {
			if len(result.ACAO) > 0 {
				break
			}
		}

		time.Sleep(time.Duration(corsmisc.Options.Delay) * time.Millisecond)

		req, err := http.NewRequest(corsmisc.Options.Method, URL, nil)
		if err != nil {
			continue
		}

		req.Header.Set("Origin", origin)

		res, err := corsmisc.Client.Do(req)
		if err != nil {
			return result, err
		}

		if res != nil {
			io.Copy(ioutil.Discard, res.Body)
			res.Body.Close()
		}

		acao := res.Header.Get("Access-Control-Allow-Origin")
		if acao == origin {
			result.ACAO = append(result.ACAO, acao)
			result.ACAC = res.Header.Get("Access-Control-Allow-Credentials")
		}
	}

	return result, err
}
