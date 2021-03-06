package main

import (
	"os"
	"github.com/spf13/viper"
)

func init() {
	port, present := os.LookupEnv("PORT")
	if !present {
		port = "8080"
	}

	viper.SetDefault("base-url", "http://127.0.0.1:" + port)
	viper.SetDefault("http_addr", ":" + port)
	viper.SetDefault("https_addr", "")
	viper.SetDefault("https_domain-whitelist", []string{"evepraisal.com"})
	viper.SetDefault("letsencrypt_email", "")
	viper.SetDefault("db_path", "db/")
	viper.SetDefault("esi_baseurl", "https://esi.tech.ccp.is/latest")
	viper.SetDefault("newrelic_app-name", "Evepraisal")
	viper.SetDefault("newrelic_license-key", "")
	viper.SetDefault("management_addr", "127.0.0.1:8090")
	viper.SetDefault("extra-js", "")
	viper.SetDefault("ad-block", "")
	viper.SetDefault("sso-authorize-url", "https://login.eveonline.com/oauth/authorize")
	viper.SetDefault("sso-token-url", "https://login.eveonline.com/oauth/token")
	viper.SetDefault("sso-verify-url", "https://login.eveonline.com/oauth/verify")

	viper.SetDefault("buyback-refine-rate",89.3)
	viper.SetDefault("buyback-reprocess-rate",55.0)
	viper.SetDefault("buyback-cap-default",101.0)
	viper.SetDefault("buyback-cap-iporg",115.0)
	viper.SetDefault("buyback-max-volume", 80000)
	viper.SetDefault("buyback-base-adjustment", 85.0)

	viper.SetDefault("adjustments", map[string]float64{})
}
