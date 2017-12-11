package esi

import (
	"fmt"
	"encoding/json"
	"bytes"
)

type CharacterAffiliation struct {
	CharacterID int64 `json:"character_id"` //character_id (integer): The character's ID ,
	CorporationID  int64 `json:"corporation_id"` //corporation_id (integer): The character's corporation ID ,
	AllianceID  int64 `json:"alliance_id,omitempty"` //alliance_id (integer, optional): The character's alliance ID, if their corporation is in an alliance ,
}

func (of *OauthFetcher) GetCharacterAffiliation(characterID int64) (*CharacterAffiliation, bool) {
	path := fmt.Sprintf("%s/characters/affiliation/", of.baseURL)
	str := fmt.Sprintf("[%d]", characterID)
	resp, err := of.client.Post(path, "application/json", bytes.NewBuffer([]byte(str)))
	if err != nil || resp.StatusCode != 200 {
		return nil, false
	}

	result := make([]CharacterAffiliation,0)
	err = json.NewDecoder(resp.Body).Decode(&result)
	defer resp.Body.Close()

	if len(result) == 0 {
		return nil, false
	}

	return &result[0], err == nil
}

