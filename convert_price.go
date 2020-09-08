package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// Doc https://www.kraken.com/features/api#get-ticker-info
type krakenResponse struct {
	Error  []string `json:"error"`
	Result struct {
		SCEUR struct {
			Ask                  []string  `json:"a"` // price, whole lot volume, lot volume
			Bid                  []string  `json:"b"` // price, whole lot volume, lot volume
			Closed               []string  `json:"c"` // price, lot volume
			Volume               []string  `json:"v"` // today, 24h
			WeightedAveragePrice []string  `json:"p"` // today, 24h
			TotalTrades          []float64 `json:"t"` // today, 24h
			Low                  []string  `json:"l"` // today, 24h
			High                 []string  `json:"h"` // today, 24h
			Opening              string    `json:"o"`
		} `json:"SCEUR"`
	} `json:"result"`
}

func getKrakenPrice() (price float64, err error) {
	resp, err := http.Get("https://api.kraken.com/0/public/Ticker?pair=SCEUR")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("status: %d", resp.StatusCode)
	}

	var kr krakenResponse
	if err = json.NewDecoder(resp.Body).Decode(&kr); err != nil {
		return 0, err
	}

	// Return the 24h average price
	return strconv.ParseFloat(kr.Result.SCEUR.Low[1], 64)
}
