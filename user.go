package evepraisal

import (
	"encoding/gob"
)

func init() {
	gob.Register(User{})
}

type User struct {
	CharacterID        int64
	CharacterName      string
	CharacterOwnerHash string
}
