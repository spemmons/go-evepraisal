package esi

import (
	"time"
	"net/http"

	"github.com/evepraisal/go-evepraisal/typedb"
	"github.com/sethgrid/pester"
	"github.com/spf13/viper"
)

type OauthFetcher struct {
	typedb  typedb.TypeDB
	client  *pester.Client
	baseURL string
}

func NewOauthFetcher(typedb typedb.TypeDB, httpClient *http.Client) *OauthFetcher {
	client := pester.NewExtendedClient(httpClient)
	client.Concurrency = 5
	client.Timeout = 30 * time.Second
	client.Backoff = pester.ExponentialJitterBackoff
	client.MaxRetries = 10
	return &OauthFetcher{typedb, client, viper.GetString("esi_baseurl")}
}

