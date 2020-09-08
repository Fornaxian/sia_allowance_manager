package main

import (
	"github.com/Fornaxian/log"
	"gitlab.com/NebulousLabs/Sia/modules"
	"gitlab.com/NebulousLabs/Sia/node/api/client"
	"gitlab.com/NebulousLabs/Sia/types"
)

func filterHosts(sia *client.Client, conf Config, scPrice float64) (err error) {
	var (
		eur              = types.SiacoinPrecision.MulFloat(1 / scPrice)
		maxStoragePrice  = eur.MulFloat(conf.MaxStoragePriceTBMonth / conf.Redundancy).Div64(1e12).Div64(blocksMonth)
		maxUploadPrice   = eur.MulFloat(conf.MaxUploadPriceTB).Div64(1e12)
		maxDownloadPrice = eur.MulFloat(conf.MaxDownloadPriceTB).Div64(1e12)
		maxCollateral    = eur.MulFloat(conf.MaxCollateralTBMonth).Div64(1e12).Div64(blocksMonth)
		maxContractPrice = eur.MulFloat(conf.MaxContractFormationPrice)
	)

	log.Debug("€1 = %s", eur.HumanString())
	log.Debug("Storage price: %s (%s)", maxStoragePrice.HumanString(), maxStoragePrice.Mul64(1e12).Mul64(blocksMonth).HumanString())
	log.Debug("Upload price: %s (%s)", maxUploadPrice.HumanString(), maxUploadPrice.Mul64(1e12).HumanString())
	log.Debug("Download price: %s (%s)", maxDownloadPrice.HumanString(), maxDownloadPrice.Mul64(1e12).HumanString())
	log.Debug("Collateral: %s (%s)", maxCollateral.HumanString(), maxCollateral.Mul64(1e12).Mul64(blocksMonth).HumanString())
	log.Debug("Contract price: %s", maxContractPrice.HumanString())
	log.Debug("")

	hosts, err := sia.HostDbAllGet()
	if err != nil {
		return err
	}

	var acceptableHosts []types.SiaPublicKey

	for _, host := range hosts.Hosts {
		if host.StoragePrice.Cmp(maxStoragePrice) == 1 {
			log.Debug(
				"Host %s rejected. Storage price of %s / TB / month too high",
				host.PublicKey.String(), host.StoragePrice.Mul64(1e12).Mul64(blocksMonth).HumanString(),
			)
			continue
		} else if host.UploadBandwidthPrice.Cmp(maxUploadPrice) == 1 {
			log.Debug(
				"Host %s rejected. Upload price of %s / TB too high",
				host.PublicKey.String(), host.UploadBandwidthPrice.Mul64(1e12).HumanString(),
			)
			continue
		} else if host.DownloadBandwidthPrice.Cmp(maxDownloadPrice) == 1 {
			log.Debug(
				"Host %s rejected. Download price of %s / TB too high",
				host.PublicKey.String(), host.DownloadBandwidthPrice.Mul64(1e12).HumanString(),
			)
			continue
		} else if host.Collateral.Cmp(maxCollateral) == 1 {
			log.Debug(
				"Host %s rejected. Collateral of %s / TB / month too high",
				host.PublicKey.String(), host.Collateral.Mul64(1e12).Mul64(blocksMonth).HumanString(),
			)
			continue
		} else if host.ContractPrice.Cmp(maxContractPrice) == 1 {
			log.Debug(
				"Host %s rejected. Contract price of %s too high",
				host.PublicKey.String(), host.ContractPrice.HumanString(),
			)
			continue
		}

		log.Debug("Host %s accepted", host.PublicKey.String())

		acceptableHosts = append(acceptableHosts, host.PublicKey)
	}

	log.Debug("%d acceptable hosts out of %d total hosts", len(acceptableHosts), len(hosts.Hosts))

	return sia.HostDbFilterModePost(modules.HostDBActiveWhitelist, acceptableHosts)
}
