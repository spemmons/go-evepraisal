package evepraisal

import (
	"github.com/evepraisal/go-evepraisal/typedb"
)

type Buyback struct {
	Totals Totals			`json:"totals"`
	Items  []AppraisalItem 	`json:"items,omitempty"`
}

func (app *App) calculateBuyback(items []AppraisalItem) (buyback Buyback) {
	buyback.Items = make([]AppraisalItem, 0)
	buybackMap := make(map[int64]*AppraisalItem)
	for _, item := range items {
		if !item.Rejected {
			app.collectBuybackItems(&buyback, buybackMap, item.TypeID, item.Quantity)
		}
	}
	app.priceAppraisalItems(buyback.Items, &buyback.Totals, "jita")
	return
}

func (app *App) collectBuybackItems(buyback *Buyback, buybackMap map[int64]*AppraisalItem, typeID int64, quantity int64) {
	t, _ := app.TypeDB.GetTypeByID(typeID)

	if t.GroupID == MineralGroupID {
		app.updateBuybackItems(buyback, buybackMap, t, quantity)
	} else {
		for _, material := range t.Materials {
			app.collectBuybackItems(buyback, buybackMap, material.TypeID, material.Quantity * quantity)
		}
	}
}

func (app *App) updateBuybackItems(buyback *Buyback, buybackMap map[int64]*AppraisalItem, t typedb.EveType, quantity int64) {
	if component, exists := buybackMap[t.ID]; exists {
		component.Quantity += quantity
	} else {
		lastItem := len(buyback.Items)
		buyback.Items = append(buyback.Items, AppraisalItem{Name: t.Name, Quantity: quantity, TypeID: t.ID, PriceAdjustment: 85})
		buybackMap[t.ID] = &buyback.Items[lastItem]
	}
}

const MineralGroupID int64 = 18

func (app *App) ableToBuyback(t typedb.EveType) bool {
	if (t.GroupID == MineralGroupID) {
		return true
	}

	for _, material := range t.Materials {
		mt, ok := app.TypeDB.GetTypeByID(material.TypeID)
		if !ok || !app.ableToBuyback(mt) {
			return false
		}
	}

	return len(t.Materials) > 0
}
