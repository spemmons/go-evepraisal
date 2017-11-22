package evepraisal

import (
	"sort"
	"github.com/evepraisal/go-evepraisal/typedb"
)

func (app *App) calculateBuyback(originalItems []AppraisalItem) (modifiedItems []AppraisalItem, buyback ItemsAndTotals) {
	buybackMap := make(map[string]*AppraisalItem)

	modifiedItems = make([]AppraisalItem,0,len(originalItems))
	for _, item := range originalItems {
		if !item.Rejected {
			itemMap := make(map[string]*AppraisalItem)
			item.Qualifier, item.Efficiency = app.collectBuybackItems(itemMap, "", 100, item.TypeID, item.Quantity)
			item.Buyback.Items = make([]AppraisalItem, 0, len(itemMap))
			for _, bbitem := range itemMap {
				item.Buyback.Items = append(item.Buyback.Items, *bbitem)
				app.updateBuybackItems(buybackMap, "", 100, bbitem.Name, bbitem.TypeID, bbitem.Quantity)
			}
			sort.Sort(ByQuantity(item.Buyback.Items))
			app.priceAppraisalItems(item.Buyback.Items, &item.Buyback.Totals, "jita")
		}
		modifiedItems = append(modifiedItems,item)
	}

	for _, bbitem := range buybackMap {
		buyback.Items = append(buyback.Items, *bbitem)
	}
	app.priceAppraisalItems(buyback.Items, &buyback.Totals, "jita")
	sort.Sort(ByQuantity(buyback.Items))
	return
}

func (app *App) collectBuybackItems(itemMap map[string]*AppraisalItem, qualifier string, efficiency float64, typeID int64, quantity int64) (string, float64) {
	t, _ := app.TypeDB.GetTypeByID(typeID)

	portion := quantity / t.PortionSize
	if t.GroupID == MineralGroupID {
		app.updateBuybackItems(itemMap, qualifier, efficiency, t.Name, t.ID, portion)
	} else {
		if t.CategoryID == AsteroidCategoryID {
			qualifier, efficiency = "REFINE",85
		} else {
			qualifier, efficiency = "REPROCESS",55
		}
		for _, material := range t.Materials {
			app.collectBuybackItems(itemMap, qualifier, efficiency, material.TypeID, portion * material.Quantity)
		}
	}

	return qualifier, efficiency
}

func (app *App) updateBuybackItems(itemMap map[string]*AppraisalItem, qualifier string, efficiency float64, typeName string, typeID int64, quantity int64) {
	quantity = int64(float64(quantity) * efficiency / 100)
	if component, exists := itemMap[typeName]; exists {
		component.Quantity += quantity
	} else {
		itemMap[typeName] = &AppraisalItem{Name: typeName, Quantity: quantity, TypeID: typeID, Efficiency: efficiency}
	}
}

const MineralGroupID int64 = 18
const AsteroidCategoryID int64 = 25

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