package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/ericlagergren/decimal"
)

func main() {
	resp, err := http.Get("https://api.fxratesapi.com/latest")
	if err != nil {
		log.Fatalf("error fetching exchange rates: %v", err)
	}
	defer resp.Body.Close()
	var fx struct {
		Success bool   `json:"success"`
		Base    string `json:"base"`
		Rates   struct {
			EUR *decimal.Big `json:"EUR"`
		} `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&fx); err != nil {
		log.Fatalf("error decoding exchange rates: %v", err)
	}
	if !fx.Success || fx.Base != "USD" {
		log.Fatal("failed to fetch exchange rates")
	}
	log.Printf("Exchange rate USD to EUR: %s", fx.Rates.EUR)
	log.Println()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	header := scanner.Text()
	if !strings.Contains(header, "Instrument") {
		log.Fatal("input does not contain 'Instrument' in the header")
	}

	var stock, stockEU, bonds, gold, cash = new(decimal.Big), new(decimal.Big),
		new(decimal.Big), new(decimal.Big), new(decimal.Big)

	for scanner.Scan() {
		instrument := strings.TrimSpace(scanner.Text())
		instrument = strings.Trim(instrument, "●")
		if instrument == "Cash Holdings" {
			break
		}
		scanner.Scan() // position
		scanner.Scan() // mkt value
		valueStr := strings.TrimSpace(scanner.Text())
		valueStr = strings.ReplaceAll(valueStr, ",", "")
		value, ok := new(decimal.Big).SetString(valueStr)
		if !ok {
			log.Fatalf("error parsing value %q", valueStr)
		}
		scanner.Scan() // price and p&l

		switch instrument {
		case "VNGA80":
			stock.Add(stock, new(decimal.Big).Mul(value, decimal.New(80, 2)))
			bonds.Add(bonds, new(decimal.Big).Mul(value, decimal.New(20, 2)))
		case "VERE", "MEUD":
			stockEU.Add(stockEU, value)
		case "VHVE", "FWRA", "VWCE":
			stock.Add(stock, value)
		case "EUNA":
			bonds.Add(bonds, value)
		case "GBSE":
			gold.Add(gold, value)
		// Betterment stock ETFs.
		case "VEA", "VTI", "ITOT", "VWO", "VTV", "SCHF", "SPYM", "SCHB", "VOE",
			"IEFA", "VBR", "IWN", "IWS", "SPDW", "IEMG", "SCHV", "SPYV", "SPEM":
			stock.Add(stock, new(decimal.Big).Mul(value, fx.Rates.EUR))
		case "XEON", "NET", "1211":
			// Not part of the rebalace.
		default:
			log.Fatalf("unexpected instrument %q", instrument)
		}
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.Contains(line, "Total Cash") {
			continue
		}
		curr, valueStr, _ := strings.Cut(line, "\t")
		valueStr = strings.TrimSpace(valueStr)
		valueStr = strings.ReplaceAll(valueStr, ",", "")
		value, ok := new(decimal.Big).SetString(valueStr)
		if !ok {
			log.Fatalf("error parsing cash value %q", valueStr)
		}
		switch strings.TrimSpace(curr) {
		case "EUR (base currency)":
			cash.Add(cash, value)
		case "USD":
			cash.Add(cash, new(decimal.Big).Mul(value, fx.Rates.EUR))
		default:
			if value.CmpAbs(decimal.New(250, 0)) > 0 {
				log.Printf("WARNING: ignoring %s %s", curr, value)
				log.Println()
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("error reading input: %v", err)
	}

	total := new(decimal.Big)
	total.Add(total, stock)
	total.Add(total, stockEU)
	total.Add(total, bonds)
	total.Add(total, gold)
	total.Add(total, cash)

	stockPct := new(decimal.Big).Quo(stock, total)
	stockEUPct := new(decimal.Big).Quo(stockEU, total)
	bondsPct := new(decimal.Big).Quo(bonds, total)
	goldPct := new(decimal.Big).Quo(gold, total)
	cashPct := new(decimal.Big).Quo(cash, total)

	log.Printf("Total stock:      %7.0f (%s)", stock, percent(stockPct))
	log.Printf("Total stock (EU): %7.0f (%s)", stockEU, percent(stockEUPct))
	log.Printf("Total bonds:      %7.0f (%s)", bonds, percent(bondsPct))
	log.Printf("Total gold:       %7.0f (%s)", gold, percent(goldPct))
	log.Printf("Total cash:       %7.0f (%s)", cash, percent(cashPct))
	log.Printf("                  -------")
	log.Printf("Total:            %7.0f", total)
	log.Println()

	targetStock := new(decimal.Big).Mul(total, decimal.New(675, 3))
	targetStockEU := new(decimal.Big).Mul(total, decimal.New(195, 3))
	targetBonds := new(decimal.Big).Mul(total, decimal.New(95, 3))
	targetGold := new(decimal.Big).Mul(total, decimal.New(35, 3))

	diffStock := new(decimal.Big).Sub(targetStock, stock)
	diffStockEU := new(decimal.Big).Sub(targetStockEU, stockEU)
	diffBonds := new(decimal.Big).Sub(targetBonds, bonds)
	diffGold := new(decimal.Big).Sub(targetGold, gold)

	log.Printf("Target stock:      %7.0f (67.5%%, %+7.0f)", targetStock, diffStock)
	log.Printf("Target stock (EU): %7.0f (19.5%%, %+7.0f)", targetStockEU, diffStockEU)
	log.Printf("Target bonds:      %7.0f ( 9.5%%, %+7.0f)", targetBonds, diffBonds)
	log.Printf("Target gold:       %7.0f ( 3.5%%, %+7.0f)", targetGold, diffGold)
	log.Println()

	// We don't sell, just buy more, so ignore negative differences and scale
	// the positive ones proportionally.
	scale := new(decimal.Big)
	for _, diff := range []*decimal.Big{diffStock, diffStockEU, diffBonds, diffGold} {
		if diff.Sign() > 0 {
			scale.Add(scale, diff)
		}
	}
	proportion := new(decimal.Big).Quo(cash, scale)
	if diffStock.Sign() > 0 {
		buyStock := new(decimal.Big).Mul(diffStock, proportion)
		log.Printf("Buy %.0f EUR of VWCE or FWRA", buyStock)
	}
	if diffStockEU.Sign() > 0 {
		buyStockEU := new(decimal.Big).Mul(diffStockEU, proportion)
		log.Printf("Buy %.0f EUR of MEUD", buyStockEU)
	}
	if diffBonds.Sign() > 0 {
		buyBonds := new(decimal.Big).Mul(diffBonds, proportion)
		log.Printf("Buy %.0f EUR of EUNA aka AGGH", buyBonds)
	}
	if diffGold.Sign() > 0 {
		buyGold := new(decimal.Big).Mul(diffGold, proportion)
		log.Printf("Buy %.0f EUR of GBSE", buyGold)
	}
}

func percent(b *decimal.Big) string {
	return fmt.Sprintf("%.2f%%", b.Mul(b, decimal.New(100, 0)))
}
