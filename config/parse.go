package config

import (
	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/viper"
	"os"
)

type Config struct {
	Account Account `yaml:"account"`
	Chain   Chain   `yaml:"chain"`
}

type Account struct {
	Mnemonic string `yaml:"mnemonic"`
	Password string `yaml:"password"`
	Key      string `yaml:"key"`
}

type Chain struct {
	rpcAddr    string `yaml:"rpc_addr"`
	GasPremium string `yaml:"gas-premium"`
	GasFeeCap  string `yaml:"gas-feecap"`
	GasLimit   int64  `yaml:"gas-limit"`
	Nonce      int    `yaml:"nonce"`
}

var (
	conf Config
	log  = logging.Logger("config")
)

func init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./conf")
	viper.AddConfigPath("../conf")
	err := viper.ReadInConfig()
	if err != nil {
		log.Errorf("ReadInConfig fail: %+v", err)
		os.Exit(1)
	}

	err = viper.Unmarshal(&conf)
	if err != nil {
		log.Errorf("Unmarshal fail: %+v", err)
		os.Exit(1)
	}
}

func Conf() Config {
	return conf
}
