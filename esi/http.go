package esi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sethgrid/pester"
)

func fetchURL(client *pester.Client, url string, r interface{}) error {
//	fmt.Printf("Fetching %s...", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
//		fmt.Printf("ERROR: %s\n", err.Error())
		return err
	}

	req.Header.Add("User-Agent", "go-evepraisal")
	resp, err := client.Do(req)
	if err != nil {
//		fmt.Printf("ERROR: %s\n", err.Error())
		return err
	}

	if resp.StatusCode != 200 {
//		fmt.Printf("INVALID STATUS: %s\n", resp.Status)
		return fmt.Errorf("Error talking to esi: %s", resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(r)
	//if err != nil {
	//	fmt.Printf("ERROR: %s\n", err.Error())
	//}

	defer resp.Body.Close()

	//fmt.Printf("DONE! %+v\n", r)
	return err
}
