package esi

import (
	"fmt"
	"time"

	"github.com/evepraisal/go-evepraisal"
	"github.com/sethgrid/pester"
	"github.com/spf13/viper"
	"log"
)

type Contract struct {
	ContractID      int64   `json:"contract_id"`                 //contract_id (integer): contract_id integer ,
	IssuerID        int64   `json:"issuer_id"`                   //issuer_id (integer): Character ID for the issuer ,
	IssuerCorpID    int64   `json:"issuer_corporation_id"`       //issuer_corporation_id (integer): Character's corporation ID for the issuer ,
	AssigneeID      int64   `json:"assignee_id"`                 //assignee_id (integer): ID to whom the contract is assigned, can be corporation or character ID ,
	AcceptorID      int64   `json:"acceptor_id"`                 //acceptor_id (integer): Who will accept the contract ,
	StartLocationID int64   `json:"start_location_id,omitempty"` //start_location_id (integer, optional): Start location ID (for Couriers contract) ,
	EndLocationID   int64   `json:"end_location_id,omitempty"`   //end_location_id (integer, optional): End location ID (for Couriers contract) ,
	Type            string  //type (string): Type of the contract = ['unknown', 'item_exchange', 'auction', 'courier', 'loan'],
	Status          string  //status (string): Status of the the contract = ['outstanding', 'in_progress', 'finished_issuer', 'finished_contractor', 'finished', 'cancelled', 'rejected', 'failed', 'deleted', 'reversed'],
	Title           string  //title (string, optional): Title of the contract ,
	ForCorp         bool    `json:"for_corporation"` //for_corporation (boolean): true if the contract was issued on behalf of the issuer's corporation ,
	Availability    string  //availability (string): To whom the contract is available = ['public', 'personal', 'corporation', 'alliance'],
	DateIssued      string  `json:"date_issued"`                //date_issued (string): Сreation date of the contract ,
	DateExpired     string  `json:"date_expired"`               //date_expired (string): Expiration date of the contract ,
	DateAccepted    string  `json:"date_accepted,omitempty"`    //date_accepted (string, optional): Date of confirmation of contract ,
	DaysToComplete  int64   `json:"days_to_complete,omitempty"` //days_to_complete (integer, optional): Number of days to perform the contract ,
	DateCompleted   string  `json:"date_completed,omitempty"`   //date_completed (string, optional): Date of completed of contract ,
	Price           float64 `json:"price,omitempty"`            //price (number, optional): Price of contract (for ItemsExchange and Auctions) ,
	Reward          float64 `json:"reward,omitempty"`           //reward (number, optional): Remuneration for contract (for Couriers only) ,
	Collateral      float64 `json:"collateral,omitempty"`       //collateral (number, optional): Collateral price (for Couriers only) ,
	Buyout          float64 `json:"buyout,omitempty"`           //buyout (number, optional): Buyout price (for Auctions only) ,
	Volume          float64 `json:"volume,omitempty"`           //volume (number, optional): Volume of items in the contract
}

type ContractStatus struct {
	Description string
	Status      string
	Contract    *Contract
}

type ContractFetcher struct {
	client  *pester.Client
	token   string
	baseURL string
}

func NewContractFetcher(token string) *ContractFetcher {
	httpClient := pester.New()
	httpClient.Concurrency = 5
	httpClient.Timeout = 30 * time.Second
	httpClient.Backoff = pester.ExponentialJitterBackoff
	httpClient.MaxRetries = 10

	return &ContractFetcher{httpClient, token, viper.GetString("esi_baseurl")}
}

func (cf *ContractFetcher) GetContracts(characterID int64) (result []Contract) {
	result = make([]Contract, 0)
	url := fmt.Sprintf("%s/characters/%d/contracts/?token=%s", cf.baseURL, characterID, cf.token)
	if err := fetchURL(cf.client, url, &result); err != nil {
		log.Printf("CONTRACT ERROR: %v\n", err)
	}
	return
}

func (cf *ContractFetcher) FindContract(user evepraisal.User, appraisalID string) *Contract {
	return cf.MatchingContract(user, appraisalID, cf.GetContracts(user.CharacterID))
}

func (cf *ContractFetcher) MatchingContract(user evepraisal.User, appraisalID string, contracts []Contract) *Contract {
	description := cf.BuybackDescription(user, appraisalID)
	for _, contract := range contracts {
		if contract.Title == description {
			return &contract
		}
	}
	return nil
}

func (cf *ContractFetcher) BuybackDescription(user evepraisal.User, appraisalID string) string {
	return fmt.Sprintf("Buyback for %v: %s", user.CharacterName, appraisalID)
}
