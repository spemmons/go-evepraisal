package esi

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/evepraisal/go-evepraisal"
	"github.com/dustin/go-humanize"
	"github.com/spf13/viper"
	"strings"
)

const ValidAssignee = 98497376      // NOTE: ID for 0.0 Massive Production
const ValidAlliance = 498125261     // NOTE: TEST
const ValidRegion = 10000039        // NOTE: Esoteria
const UnrestrictedSystem = 30003144 // NOTE: H-T40Z

type Contract struct {
	ContractID      int64   `json:"contract_id"`                 //contract_id (integer): contract_id integer ,
	IssuerID        int64   `json:"issuer_id"`                   //issuer_id (integer): Character ID for the issuer ,
	IssuerCorpID    int64   `json:"issuer_corporation_id"`       //issuer_corporation_id (integer): Character's corporation ID for the issuer ,
	AssigneeID      int64   `json:"assignee_id"`                 //assignee_id (integer): ID to whom the contract is assigned, can be corporation or character ID ,
	AcceptorID      int64   `json:"acceptor_id"`                 //acceptor_id (integer): Who will accept the contract ,
	StartLocationID int64   `json:"start_location_id,omitempty"` //start_location_id (integer, optional): Start location ID (for Couriers contract) ,
	EndLocationID   int64   `json:"end_location_id,omitempty"`   //end_location_id (integer, optional): End location ID (for Couriers contract) ,
	Type            string                                       //type (string): Type of the contract = ['unknown', 'item_exchange', 'auction', 'courier', 'loan'],
	Status          string                                       //status (string): Status of the the contract = ['outstanding', 'in_progress', 'finished_issuer', 'finished_contractor', 'finished', 'cancelled', 'rejected', 'failed', 'deleted', 'reversed'],
	Title           string                                       //title (string, optional): Title of the contract ,
	ForCorp         bool    `json:"for_corporation"`             //for_corporation (boolean): true if the contract was issued on behalf of the issuer's corporation ,
	Availability    string                                       //availability (string): To whom the contract is available = ['public', 'personal', 'corporation', 'alliance'],
	DateIssued      string  `json:"date_issued"`                 //date_issued (string): Сreation date of the contract ,
	DateExpired     string  `json:"date_expired"`                //date_expired (string): Expiration date of the contract ,
	DateAccepted    string  `json:"date_accepted,omitempty"`     //date_accepted (string, optional): Date of confirmation of contract ,
	DaysToComplete  int64   `json:"days_to_complete,omitempty"`  //days_to_complete (integer, optional): Number of days to perform the contract ,
	DateCompleted   string  `json:"date_completed,omitempty"`    //date_completed (string, optional): Date of completed of contract ,
	Price           float64 `json:"price,omitempty"`             //price (number, optional): Price of contract (for ItemsExchange and Auctions) ,
	Reward          float64 `json:"reward,omitempty"`            //reward (number, optional): Remuneration for contract (for Couriers only) ,
	Collateral      float64 `json:"collateral,omitempty"`        //collateral (number, optional): Collateral price (for Couriers only) ,
	Buyout          float64 `json:"buyout,omitempty"`            //buyout (number, optional): Buyout price (for Auctions only) ,
	Volume          float64 `json:"volume,omitempty"`            //volume (number, optional): Volume of items in the contract

	SystemID     int64  `json:"system_id,omitempty"`     // NOTE - not part of API structure
	LocationName string `json:"location_name,omitempty"` // NOTE - not part of API structure
}

type ContractItem struct {
	RecordID    int64 `json:"record_id"`              //record_id (integer): Unique ID for the item ,
	TypeID      int64 `json:"type_id"`                //type_id (integer): Type ID for item ,
	Quantity    int64 `json:"quantity"`               //quantity (integer): Number of items in the stack ,
	RawQuantity int64 `json:"raw_quantity,omitempty"` //raw_quantity (integer, optional): -1 indicates that the item is a singleton (non-stackable). If the item happens to be a Blueprint, -1 is an Original and -2 is a Blueprint Copy ,
	IsSingleton bool  `json:"is_singleton"`           //is_singleton (boolean): is_singleton boolean ,
	IsIncluded  bool  `json:"is_included"`            //is_included (boolean): true if the contract issuer has submitted this item with the contract, false if the isser is asking for this item in the contract.
}

type ContractStatus struct {
	Title    string
	Summary  string
	Contract *Contract
	Errors   []string
}

func (of *OauthFetcher) GetContracts(characterID int64) (result []Contract, err error) {
	result = make([]Contract, 0)
	url := fmt.Sprintf("%s/characters/%d/contracts/", of.baseURL, characterID)
	err = fetchURL(of.client, url, &result)
	return
}

func (of *OauthFetcher) GetContractItems(characterID int64, contractID int64) (result []ContractItem, err error) {
	result = make([]ContractItem, 0)
	url := fmt.Sprintf("%s/characters/%d/contracts/%d/items/", of.baseURL, characterID, contractID)
	err = fetchURL(of.client, url, &result)
	return
}

func (of *OauthFetcher) GetContractStatus(user *evepraisal.User, appraisal *evepraisal.Appraisal) *ContractStatus {
	contracts, err := of.GetContracts(user.CharacterID)
	if err != nil {
		fmt.Printf("CONTRACT ERROR: ", err)
		return &ContractStatus{of.BuybackTitle(user, appraisal.ID), "error", nil, []string{err.Error()}}
	}

	return of.EvaluateContract(user, appraisal, contracts)
}

func (of *OauthFetcher) EvaluateContract(user *evepraisal.User, appraisal *evepraisal.Appraisal, contracts []Contract) *ContractStatus {
	title := of.BuybackTitle(user, appraisal.ID)

	var summary = "not_found"
	errors := []string{}
	for _, item := range appraisal.Original.Items {
		if item.Rejected {
			errors = append(errors, fmt.Sprintf("%s cannot be included, please remove it before submitting a buyback contract", item.DisplayName()))
			summary = "invalid"
		}
	}

	var contract *Contract
	if summary != "invalid" {
		contract = of.findMatchingContract(appraisal.ID, contracts)
		if contract != nil {
			summary = contract.Status
			title = contract.Title

			errors = of.validateContract(user, appraisal, contract)

			if summary == "outstanding" {
				if len(errors) > 0 {
					summary = "invalid"
				} else {
					summary = "valid"
				}
			}
		}
	}

	return &ContractStatus{title, summary, contract, errors}
}

func (of *OauthFetcher) findMatchingContract(appraisalID string, contracts []Contract) *Contract {
	for _, contract := range contracts {
		if strings.Contains(contract.Title, appraisalID) {
			return &contract
		}
	}
	return nil
}

func (of *OauthFetcher) BuybackTitle(user *evepraisal.User, appraisalID string) string {
	return fmt.Sprintf("Buyback for %v: %s", user.CharacterName, appraisalID)
}

func (of *OauthFetcher) validateContract(user *evepraisal.User, appraisal *evepraisal.Appraisal, contract *Contract) (errors []string) {
	errors = []string{}

	title := of.BuybackTitle(user, appraisal.ID)
	if contract.Title != title {
		errors = append(errors, fmt.Sprintf("Contract title does not exactly match '%s'", title))
	}
	if contract.Type != "item_exchange" {
		errors = append(errors, "Contract Type must be an 'Item Exchange'")
	}
	if contract.Availability != "personal" {
		errors = append(errors, "Contract Availability must be 'Private'")
	}
	if contract.AssigneeID != ValidAssignee {
		errors = append(errors, "Contract Assignee must be '0.0 Massive Production' (ticker 0MP)")
	}
	if contract.Reward != 0 {
		errors = append(errors, "Contract Reward must be 0 isk")
	}
	if contract.Price != appraisal.BuybackOffer() {
		errors = append(errors, "Contract Price must equal Buyback Offer")
	}
	if math.Abs(contract.Volume-appraisal.Original.Totals.Volume) >= 0.01 {
		errors = append(errors, fmt.Sprintf("Expected volume of %v but found %v", appraisal.Original.Totals.Volume, contract.Volume))
	}

	errors = append(errors, validateContractDuration(contract)...)
	if len(errors) > 0 {
		return
	}

	errors = append(errors, of.validateContractLocation(contract)...)
	if len(errors) > 0 {
		return
	}

	errors = append(errors, of.validateContractItems(user, appraisal, contract)...)
	if len(errors) > 0 {
		return
	}

	if contract.SystemID != UnrestrictedSystem && !appraisal.OnlyCompressedOre() {
		buybackMaxVolume := viper.GetFloat64("buyback-max-volume")
		if appraisal.Original.Totals.Volume > buybackMaxVolume {
			errors = append(errors, fmt.Sprintf("Buyback volume of %s m3 is too large; it must be no larger than %s m3", humanize.Commaf(appraisal.Original.Totals.Volume), humanize.Commaf(buybackMaxVolume)))
		}
	}

	return
}

func (of *OauthFetcher) validateContractItems(user *evepraisal.User, appraisal *evepraisal.Appraisal, contract *Contract) (errors []string) {
	errors = make([]string, 0)
	contractItems, err := of.GetContractItems(user.CharacterID, contract.ContractID)
	if err != nil {
		errors = append(errors, err.Error())
	} else {
		for _, expectedItem := range appraisal.Original.Items {
			var contractQuantity int64
			for _, contractItem := range contractItems {
				if expectedItem.TypeID == contractItem.TypeID {
					contractQuantity += contractItem.Quantity
				}
			}

			if contractQuantity != expectedItem.Quantity {
				errors = append(errors, fmt.Sprintf("Expected %d units of %s but found %d", expectedItem.Quantity, expectedItem.DisplayName(), contractQuantity))
			}
		}

		var unexpectedItems = map[int64]bool{}
		for _, contractItem := range contractItems {
			var found bool
			for _, expectedItem := range appraisal.Original.Items {
				if contractItem.TypeID == expectedItem.TypeID {
					found = true
					break
				}
			}

			if !found {
				unexpectedItems[contractItem.TypeID] = true
			}
		}

		for typeID, _ := range unexpectedItems {
			evetype, found := of.typedb.GetTypeByID(typeID)
			if found {
				errors = append(errors, fmt.Sprintf("Unexpected %s found in contract", evetype.Name))
			} else {
				errors = append(errors, "Unknown type found in contract")
			}
		}
	}

	if len(errors) > 0 {
		log.Printf("Invalid contract for appraisal %s\n", appraisal.ID)
		for _, err := range errors {
			log.Printf("- %s\n", err)
		}
	}

	return
}

func (of *OauthFetcher) validateContractLocation(contract *Contract) (errors []string) {
	systemID, name, found := of.FindLocation(contract.StartLocationID)
	if !found {
		errors = append(errors, "Contract location cannot be found")
		return
	}

	contract.SystemID = systemID
	contract.LocationName = name

	if regionID, _ := of.FindRegionForSystemID(systemID); regionID != ValidRegion {
		errors = append(errors, "Contract must be in Esoteria")
		return
	}

	if allianceID, _ := of.FindAllianceForSystemID(systemID); allianceID != ValidAlliance {
		errors = append(errors, "Contract must be in a system controlled by TEST")
		return
	}

	return
}

func validateContractDuration(contract *Contract) (errors []string) {
	issueDate, errors := parseDateString(contract.DateIssued)
	if len(errors) > 0 {
		return
	}

	expireDate, errors := parseDateString(contract.DateExpired)
	if len(errors) > 0 {
		return
	}

	if expireDate.Sub(issueDate).Hours() < 2*7*24 {
		errors = append(errors, "Contract Duration must be a minimum of 2 weeks")
	}

	return
}

func parseDateString(value string) (result time.Time, errors []string) {
	errors = []string{}
	result, err := time.Parse(time.RFC3339, value)
	if err != nil {
		errors = []string{err.Error()}
	}
	return
}
