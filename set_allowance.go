package main

import (
	"fmt"

	"github.com/Fornaxian/log"
	"gitlab.com/NebulousLabs/Sia/node/api/client"
	"gitlab.com/NebulousLabs/Sia/types"
)

const (
	blocksDay   = 144
	blocksWeek  = blocksDay * 7
	blocksYear  = uint64(blocksDay * 365.25) // I tried 365.2425 days but it causes problems
	blocksMonth = uint64(blocksYear / 12)
)

func adjustAllowance(sia *client.Client, conf Config, scPrice float64) (err error) {
	// Get unspent unallocated funds
	rg, err := sia.RenterGet()
	if err != nil {
		return fmt.Errorf("Could not get renter metrics: %w", err)
	}

	fm := rg.FinancialMetrics

	funds := rg.Settings.Allowance.Funds
	if funds.IsZero() {
		funds = types.SiacoinPrecision.Mul64(500) // Start with 500 SC
	}

	spent := fm.ContractFees.
		Add(fm.UploadSpending).
		Add(fm.DownloadSpending).
		Add(fm.StorageSpending)

	// Calculate unspent allocated
	unspentAllocated := types.ZeroCurrency
	if fm.TotalAllocated.Cmp(spent) >= 0 {
		unspentAllocated = fm.TotalAllocated.Sub(spent)
	}

	// Calculate unspent unallocated
	unspentUnallocated := types.ZeroCurrency
	if fm.Unspent.Cmp(unspentAllocated) >= 0 {
		unspentUnallocated = fm.Unspent.Sub(unspentAllocated)
	}

	var (
		lowBound     = funds.Div64(10) // 10%
		highBound    = funds.Div64(4)  // 25%
		adjustMargin = funds.Div64(10) // 10%
	)

	log.Debug(
		"Unallocated lower bound: %s, current: %s, higher bound: %s",
		lowBound.HumanString(), unspentUnallocated.HumanString(), highBound.HumanString(),
	)

	// If the unspent unallocated funds are less than 10% of the total funds we
	// increase the allowance by 10%. We set the expected storage / upload /
	// download size based on the Siacoin price
	if unspentUnallocated.Cmp(lowBound) <= 0 {
		// Unallocated funds are less than 10% of the total allowance. We need
		// to increase it to continue uploading. Here we increase the allowance
		// by 10%

		funds = funds.Add(adjustMargin)
		log.Debug("Funds too low. Increasing to: %s", funds.HumanString())
	} else if unspentUnallocated.Cmp(highBound) >= 0 {
		// Unallocated funds are more than 25% of the allowance. We need to
		// decrease it to save money on fees. Here we lower the allowance by 10%

		funds = funds.Sub(adjustMargin)
		log.Debug("Funds too high. Decreasing to: %s", funds.HumanString())
	} else {
		log.Debug(
			"Enough margin left. No need to increase allowance. Funds: %s Unspent unallocated: %s",
			funds.HumanString(), unspentUnallocated.HumanString(),
		)

		return nil
	}

	fundsSC, _ := funds.Div64(1e12).Float64()
	fundsSC /= 1e12
	var (
		fundsEUR         = fundsSC * scPrice
		allowanceMonths  = float64(rg.Settings.Allowance.Period) / float64(blocksMonth)
		fundsEURPerMonth = fundsEUR / allowanceMonths
		expectedStorage  = fundsEURPerMonth / conf.MaxStoragePriceTBMonth * 1e12
		expectedUpload   = expectedStorage * 0.1
		expectedDownload = expectedStorage * 0.2
	)

	log.Debug(
		"SC %.3f, €%.3f, months %.3f, €/month %.3f, storage %v, upload %v, download %v",
		fundsSC,
		fundsEUR,
		allowanceMonths,
		fundsEURPerMonth,
		FormatData(int64(expectedStorage)),
		FormatData(int64(expectedUpload)),
		FormatData(int64(expectedDownload)),
	)

	// Set the allowance using the calculated values
	// return nil
	return sia.RenterPostPartialAllowance().
		WithExpectedStorage(uint64(expectedStorage)).
		WithExpectedUpload(uint64(expectedUpload / float64(rg.Settings.Allowance.Period))).
		WithExpectedDownload(uint64(expectedDownload / float64(rg.Settings.Allowance.Period))).
		WithExpectedRedundancy(conf.Redundancy).
		WithHosts(conf.Hosts).
		WithFunds(funds).Send()
}