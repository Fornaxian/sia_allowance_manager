package main

import (
	"fmt"

	"github.com/Fornaxian/config"
	"gitlab.com/NebulousLabs/Sia/node/api/client"
)

type Config struct {
	SiaAPIPassword string `toml:"sia_api_password"`

	// This price includes redundancy. This is what the user ends up paying
	MaxStoragePriceTBMonth    float64 `toml:"max_storage_price_tb_month"`
	MaxDownloadPriceTB        float64 `toml:"max_download_price_tb"`
	MaxUploadPriceTB          float64 `toml:"max_upload_price_tb"`
	MaxContractFormationPrice float64 `toml:"max_contract_formation_price"`

	// This price is without redundancy. The collateral price is calculated per
	// host
	MaxCollateralTBMonth float64 `toml:"max_collateral_tb_month"`
	Redundancy           float64 `toml:"redundancy"`
	Hosts                uint64  `toml:"hosts"`
}

const defaultConf = `# Allowance manager configuration

# Password for accessing the Sia API. Can be found in ~/.sia/apipassword
sia_api_password = ""

# Max storage price in euros per month per TB including redundancy.
max_storage_price_tb_month = 4.50

# Max download price in euros per TB
max_download_price_tb = 1.00

# Max upload price in euros per TB
max_upload_price_tb = 0.50

# Max contract formation fee in euros per contract
max_contract_formation_price = 0.01

# Max collateral price in euros per month per terabyte. This value is per host.
# So without redundancy
max_collateral_tb_month = 5.00

# Default redundancy value to use in calculations
redundancy = 3.00

# Number of hosts to use when creating contracts
hosts = 50
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
