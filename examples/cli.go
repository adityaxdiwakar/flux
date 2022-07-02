package main

import (
	"fmt"
	"log"
	"os"

	"github.com/adityaxdiwakar/flux"
	"github.com/adityaxdiwakar/tda-go"
)

func main() {
	tdaSession := tda.Session{
		Refresh:     "<YOUR_REFRESH_TOKEN>",
		ConsumerKey: "<YOUR_CONSUMER_KEY>",
		RootUrl:     "https://api.tdameritrade.com/v1",
	}

	s, err := flux.New(tdaSession)
	if err != nil {
		log.Fatal(err)
	}
	s.Open()

	for {
		var input string
		fmt.Printf("Please enter a ticker (or type exit): ")
		fmt.Scanln(&input)
		// if the input is exit or empty, close the session and exit the program
		if input == "exit" || input == "" {
			s.Close()
			os.Exit(1)
		} else {
			payload, err := s.RequestChart(flux.ChartRequestSignature{Ticker: input, Width: "HOUR1", Range: "DAY1"})
			if err != nil {
				fmt.Printf("Could not load ticker, %v\n", err)
			} else {
				fmt.Printf("%s\n", payload.Instrument.Description)
			}
		}
	}
}
