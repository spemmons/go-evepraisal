package esi

import (
	"fmt"
	"time"

	"github.com/evepraisal/go-evepraisal"
	"github.com/sethgrid/pester"
	"github.com/spf13/viper"
	"github.com/evepraisal/go-evepraisal/typedb"
)

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

type ContractFetcher struct {
	typedb  typedb.TypeDB
	client  *pester.Client
	token   string
	baseURL string
}

func NewContractFetcher(typedb typedb.TypeDB, token string) *ContractFetcher {
	httpClient := pester.New()
	httpClient.Concurrency = 5
	httpClient.Timeout = 30 * time.Second
	httpClient.Backoff = pester.ExponentialJitterBackoff
	httpClient.MaxRetries = 10

	return &ContractFetcher{typedb,httpClient, token, viper.GetString("esi_baseurl")}
}

func (cf *ContractFetcher) GetContracts(characterID int64) (result []Contract, err error) {
	result = make([]Contract, 0)
	url := fmt.Sprintf("%s/characters/%d/contracts/?token=%s", cf.baseURL, characterID, cf.token)
	err = fetchURL(cf.client, url, &result)
	return
}

func (cf *ContractFetcher) GetContractItems(characterID int64, contractID int64) (result []ContractItem, err error) {
	result = make([]ContractItem, 0)
	url := fmt.Sprintf("%s/characters/%d/contracts/%d/items/?token=%s", cf.baseURL, characterID, contractID, cf.token)
	err = fetchURL(cf.client, url, &result)
	return
}

func (cf *ContractFetcher) GetContractStatus(user *evepraisal.User, appraisal *evepraisal.Appraisal) *ContractStatus {
	contracts, err := cf.GetContracts(user.CharacterID)
	if err != nil {
		fmt.Printf("CONTRACT ERROR: ", err)
		return &ContractStatus{cf.BuybackTitle(user, appraisal.ID), "error", nil, []string{err.Error()}}
	}

	return cf.EvaluateContract(user, appraisal, contracts)
}

const ValidAssignee = 98497376 // NOTE: ID for 0.0 Massive Production

func (cf *ContractFetcher) EvaluateContract(user *evepraisal.User, appraisal *evepraisal.Appraisal, contracts []Contract) *ContractStatus {
	title := cf.BuybackTitle(user, appraisal.ID)

	var summary = "not_found"
	errors := []string{}
	for _, item := range appraisal.Original.Items {
		if item.Rejected {
			errors = append(errors, fmt.Sprintf("%s cannot be included, please remove it before submitting a buyback contract", item.DisplayName()))
			summary = "invalid"
		}
	}

	contract := cf.MatchingContract(title, contracts)
	if contract != nil {
		summary = contract.Status
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
		if contract.Volume != appraisal.Original.Totals.Volume {
			errors = append(errors, fmt.Sprintf("Expected volume of %v but found %v", appraisal.Original.Totals.Volume, contract.Volume))
		}
		// TODO check required duration
		// TODO check TEST system

		contractItems, err := cf.GetContractItems(user.CharacterID, contract.ContractID)
		if err != nil {
			errors = append(errors, err.Error())
		} else {
			for _, expectedItem := range appraisal.Original.Items {
				var contractQuantity int64
				for _, contractItem := range contractItems {
					if expectedItem.TypeID  == contractItem.TypeID {
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
				evetype, found := cf.typedb.GetTypeByID(typeID)
				if found {
					errors = append(errors, fmt.Sprintf("Unexpected %s found in contract", evetype.Name))
				} else {
					errors = append(errors, "Unknown type found in contract")
				}
			}
		}

		if summary == "outstanding" {
			if len(errors) > 0 {
				summary = "invalid"
			} else {
				summary = "valid"
			}
		}
	}

	return &ContractStatus{title, summary, contract, errors}
}

func (cf *ContractFetcher) MatchingContract(title string, contracts []Contract) *Contract {
	for _, contract := range contracts {
		if contract.Title == title {
			return &contract
		}
	}
	return nil
}

func (cf *ContractFetcher) BuybackTitle(user *evepraisal.User, appraisalID string) string {
	return fmt.Sprintf("Buyback for %v: %s", user.CharacterName, appraisalID)
}
