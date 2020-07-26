# flux
![Flux Logo](https://i.imgur.com/MFQBlUd.png)

Flux is a highly performant and error-redundant wrapper for the TDAmeritrade Provisioner

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/adityaxdiwakar/flux)
![Go Version](https://img.shields.io/github/go-mod/go-version/adityaxdiwakar/flux?style=flat-square)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square)](https://opensource.org/licenses/MIT)
![Code Size](https://img.shields.io/github/languages/code-size/adityaxdiwakar/flux?style=flat-square)
[![Go Report Card](https://goreportcard.com/badge/github.com/adityaxdiwakar/flux?style=flat-square)](https://goreportcard.com/report/github.com/adityaxdiwakar/flux)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fadityaxdiwakar%2Fflux.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fadityaxdiwakar%2Fflux?ref=badge_shield)

## Usage
Flux works in unison with my TDAmeritrade API Wrapper [tda-go](https://github.com/adityaxdiwakar/tda-go) for the authentication and access token functions. This is why this is also required to be imported (and potentially added to your ``go.mod`` file).

```go
import "github.com/adityaxdiwakar/flux"
```

This program is intended to act as a non-blocking interface to TDAmeritrade data provisioner, it is required to be instantiated and opened, but operaes within a goroutine to prevent blocking the main process. 

A simple example: 
```go
package main

import (
  "fmt"
  "os"
  "syscall"
   
  "github.com/adityaxdiwakar/flux"
  "github.com/adityaxdiwakar/tda-go"
)

func main() {
  tdaSession := tda.Session{
     Refresh: "<YOUR_REFRESH_TOKEN>,
     ConsumerKey: "<YOUR_CONSUMER_KEY>",
     RootUrl: "https://api.tdameritrade.com/v1",
     
  }
  
  s, err := flux.New(tdaSession)
  if err != nil {
     panic(err) // panic if an error is present
  }
  s.Open()
  
  payload, err := s.RequestChart(flux.ChartRequestSignature{
    Ticker: "AAPL", Width: "HOUR1", Range: "DAY1",
  })
  fmt.Println(payload.Instrument.Description) // prints `APPLE INC COM`
  
  // make a channel that is notified of any system interrupts 
  // this is a blocking set of lines, if the `sc` channel is notified
  // the function will continue and move to s.Close()
  sc := make(chan os.signal, 1)
  signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
  <-sc
  
  s.Close()
}




```

There are more examples in the [examples folder](examples/).


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fadityaxdiwakar%2Fflux.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fadityaxdiwakar%2Fflux?ref=badge_large)
