package typedb

type TypeDB interface {
	GetType(typeName string) (EveType, bool)
	HasType(typeName string) bool
	GetTypeByID(typeID int64) (EveType, bool)
	PutType(EveType) error
	Search(s string) []EveType
	Delete() error
	Close() error
}

type EveType struct {
	ID                int64       `json:"id"`
	GroupID           int64       `json:"group_id"`
	MarketGroupID     int64       `json:"market_group_id"`
	CategoryID		  int64		  `json:"category_id"`
	Name              string      `json:"name"`
	Volume            float64     `json:"volume"`
	PackagedVolume    float64     `json:"packaged_volume"`
	BasePrice         float64     `json:"base_price"`
	PortionSize		  int64	      `json:"portion_size"`
	BlueprintProducts []Component `json:"blueprint_products,omitempty"`
	Components        []Component `json:"components,omitempty"`
	BaseComponents    []Component `json:"base_components,omitempty"`
	Materials		  []Component `json:"materials,omitempty"`
}

type Component struct {
	Quantity int64 `json:"quantity"`
	TypeID   int64 `json:"type_id"`
}
