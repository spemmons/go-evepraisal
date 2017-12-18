package discord

import (
	"bytes"
	"fmt"

	"github.com/spf13/viper"
	"github.com/sethgrid/pester"
)

func PostMessage(message string) {
	httpClient := pester.New()

	url := viper.GetString("discord-webhook")
	if url != "" {
		body := fmt.Sprintf("{\"content\":\"%s\"}", message)
		resp, err := httpClient.Post(url, "application/json", bytes.NewBuffer([]byte(body)))
		if err != nil || resp.StatusCode != 204 {
			fmt.Printf("DISCORD ERROR(%v): %v\n", resp.StatusCode, err)
		}
	}
}