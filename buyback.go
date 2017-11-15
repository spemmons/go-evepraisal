package evepraisal

import (
	"github.com/evepraisal/go-evepraisal/typedb"
	"sort"
	"fmt"
)

type Buyback struct {
	Totals Totals			`json:"totals"`
	Items  []AppraisalItem 	`json:"items,omitempty"`
}

func (app *App) calculateBuyback(items []AppraisalItem) (buyback Buyback) {
	buybackMap := make(map[string]*AppraisalItem)
	for _, item := range items {
		if !item.Rejected {
			app.collectBuybackItems(buybackMap, "", 100, item.TypeID, item.Quantity)
		}
	}

	buyback.Items = make([]AppraisalItem, 0, len(buybackMap))
	for _, item := range buybackMap {
		buyback.Items = append(buyback.Items, *item)
	}

	sort.Sort(ByQuantity(buyback.Items))

	app.priceAppraisalItems(buyback.Items, &buyback.Totals, "jita")
	return
}

func (app *App) collectBuybackItems(buybackMap map[string]*AppraisalItem, qualifier string, adjustment float64, typeID int64, quantity int64) {
	t, _ := app.TypeDB.GetTypeByID(typeID)

	if t.GroupID == MineralGroupID {
		app.updateBuybackItems(buybackMap, qualifier, adjustment, t, quantity)
	} else {
		fmt.Printf("Q: %v TYPE: %+v", qualifier, t)
		for _, material := range t.Materials {
			app.collectBuybackItems(buybackMap, " (refined)", 85, material.TypeID, material.Quantity * quantity)
		}
	}
}

func (app *App) updateBuybackItems(buybackMap map[string]*AppraisalItem, qualifier string, adjustment float64, t typedb.EveType, quantity int64) {
	buybackKey := t.Name + qualifier
	if component, exists := buybackMap[buybackKey]; exists {
		component.Quantity += quantity
	} else {
		buybackMap[buybackKey] = &AppraisalItem{Name: buybackKey, Quantity: quantity, TypeID: t.ID, PriceAdjustment: adjustment}
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

type ByQuantity []AppraisalItem

func (a ByQuantity) Len() int           { return len(a) }
func (a ByQuantity) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByQuantity) Less(i, j int) bool { return a[i].Quantity > a[j].Quantity }