# corsmisc

![made with go](https://img.shields.io/badge/made%20with-Go-0040ff.svg) ![Maintenance](https://img.shields.io/badge/maintained%3F-yes-0040ff.svg) [![open issues](https://img.shields.io/github/issues-raw/drsigned/corsmisc.svg?style=flat&color=0040ff)](https://github.com/drsigned/corsmisc/issues?q=is:issue+is:open) [![closed issues](https://img.shields.io/github/issues-closed-raw/drsigned/corsmisc.svg?style=flat&color=0040ff)](https://github.com/drsigned/corsmisc/issues?q=is:issue+is:closed) [![license](https://img.shields.io/badge/license-MIT-gray.svg?colorB=0040FF)](https://github.com/drsigned/corsmisc/blob/master/LICENSE) [![twitter](https://img.shields.io/badge/twitter-@drsigned-0040ff.svg)](https://twitter.com/drsigned)

corsmisc is a tool designed to discover CORS misconfigurations vulnerabilities.

## Resources

* [Installation](#installation)
    * [From Binary](#from-binary)
    * [From source](#from-source)
    * [From github](#from-github)
* [Usage](#usage)
* [Contribution](#contribution)

## Installation

#### From Binary

You can download the pre-built binary for your platform from this repository's [releases](https://github.com/drsigned/corsmisc/releases/) page, extract, then move it to your `$PATH`and you're ready to go.

#### From Source

corsmisc requires **go1.14+** to install successfully. Run the following command to get the repository:

```bash
$ go get github.com/drsigned/corsmisc/cmd/corsmisc
```

#### From Github

```bash
git clone https://github.com/drsigned/corsmisc.git; cd corsmisc/cmd/corsmisc; go build; mv corsmisc /usr/local/bin/; corsmisc -h
```

## Usage

To display help message for sigurlx use the `-h` flag:

```
$ corsmisc -h

                              _
  ___ ___  _ __ ___ _ __ ___ (_)___  ___
 / __/ _ \| '__/ __| '_ ` _ \| / __|/ __|
| (_| (_) | |  \__ \ | | | | | \__ \ (__
 \___\___/|_|  |___/_| |_| |_|_|___/\___| v1.2.0

USAGE:
  corsmisc [OPTIONS]

OPTIONS:
  -all            test all Origin's
  -c              concurrency level (default: 50)
  -d              delay between requests (default: 100ms)
  -nc             no color mode
  -o              JSON output file
  -s              silent mode
  -timeout        HTTP request timeout (default: 10s)
  -urls           list of urls (use `-` to read stdin)
  -UA             HTTP user agent
  -v              verbose mode
  -X              HTTP method to use (default: GET)
  -x              HTTP Proxy URL
```

## Contribution

[Issues](https://github.com/drsigned/corsmisc/issues) and [Pull Requests](https://github.com/drsigned/corsmisc/pulls) are welcome.