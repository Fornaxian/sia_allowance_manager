package main

import (
	"fmt"

	"github.com/Fornaxian/config"
	"gitlab.com/NebulousLabs/Sia/node/api/client"
)

type Config struct {
	SiaAPIPassword string `toml:"sia_api_password"`

	// These prices are per host, without redundancy. Multiply the storage and
	// collateral values by the Redundancy float to get the prices which the
	// renter ends up paying
	MaxStoragePriceTBMonth    float64 `toml:"max_storage_price_tb_month"`
	MaxDownloadPriceTB        float64 `toml:"max_download_price_tb"`
	MaxUploadPriceTB          float64 `toml:"max_upload_price_tb"`
	MaxContractFormationPrice float64 `toml:"max_contract_formation_price"`
	MaxCollateralTBMonth      float64 `toml:"max_collateral_tb_month"`
	Redundancy                float64 `toml:"redundancy"`
	Hosts                     uint64  `toml:"hosts"`
	ContractLength            uint64  `toml:"contract_length"`
	RenewWindow               uint64  `toml:"renew_window"`
}

const defaultConf = `# Allowance manager configuration

# Password for accessing the Sia API. Can be found in ~/.sia/apipassword
sia_api_password = ""

# Max storage price in euros per month per TB per host (without redundancy)
max_storage_price_tb_month = 1.80

# Max download price in euros per TB
max_download_price_tb = 2.50

# Max upload price in euros per TB
max_upload_price_tb = 2.00

# Max contract formation fee in euros per contract
max_contract_formation_price = 0.10

# Max collateral price in euros per month per TB per host (without redundancy)
max_collateral_tb_month = 8.00

# Data redundancy value to use in calculations. When setting the allowance the
# max_storage_price_tb_month and max_collateral_tb_month will be multiplied by
# this value in order to calculate the real price you will end up paying
redundancy = 3.00

# Number of hosts to use when creating contracts
hosts = 50

# Contract length and renew window in blocks. The defaults are 3 months and one
# month, respectively. Assuming a month is exactly 30 days, which it isn't. But
# it's close enough
contract_length = 12960
renew_window = 4320
`

func main() {
	var err error
	var conf Config
	if _, err = config.New(defaultConf, "", "sia_allowance_manager.toml", &conf, true); err != nil {
		panic(err)
	}

	sia := client.New(client.Options{
		Address:  "127.0.0.1:9980",
		Password: conf.SiaAPIPassword,
	})

	scPrice, err := getKrakenPrice()
	if err != nil {
		panic(err)
	}

	if err = filterHosts(sia, conf, scPrice); err != nil {
		panic(err)
	}

	if err = adjustAllowance(sia, conf, scPrice); err != nil {
		panic(err)
	}
}

// FormatData prints an amount if bytes in a readable rounded amount
func FormatData(size int64) string {
	var fmtSize = func(n float64, u string) string {
		var f string
		if n >= 100 {
			f = "%.1f"
		} else if n >= 10 {
			f = "%.2f"
		} else {
			f = "%.3f"
		}
		return fmt.Sprintf(f+" "+u, n)
	}
	if size >= 1e18 {
		return fmtSize(float64(size)/1e12, "EB")
	} else if size >= 1e15 {
		return fmtSize(float64(size)/1e12, "PB")
	} else if size >= 1e12 {
		return fmtSize(float64(size)/1e12, "TB")
	} else if size >= 1e9 {
		return fmtSize(float64(size)/1e9, "GB")
	} else if size >= 1e6 {
		return fmtSize(float64(size)/1e6, "MB")
	} else if size >= 1e3 {
		return fmtSize(float64(size)/1e3, "kB")
	}
	return fmt.Sprintf("%d B", size)
}
