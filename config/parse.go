package config

import (
	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Account Account `yaml:"account"`
	Chain   Chain   `yaml:"chain"`
}

type Account struct {
	Mnemonic  string `yaml:"mnemonic"`
	Password  bool   `yaml:"password"`
	Key       string `yaml:"key"`
	KeyFormat string `yaml:"keyFormat"`
}

type Chain struct {
	MaxFee   string `json:"maxFee"`
	RpcAddr  string `yaml:"rpcAddr"`
	Token    string `json:"token"`
	Explorer string `yaml:"explorer"`
}

var (
	conf Config
	log  = logging.Logger("config")
)

func InitConfig(confPath string) {
	if confPath != "" {
		log.Infow("load config", "path", confPath)

		confName := strings.Split(filepath.Base(confPath), ".")

		viper.SetConfigName(confName[0])
		viper.SetConfigType("yaml")
		viper.AddConfigPath(filepath.Dir(confPath))
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./conf")
		viper.AddConfigPath("../conf")
		viper.AddConfigPath("./")
		viper.AddConfigPath("../")
	}

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
