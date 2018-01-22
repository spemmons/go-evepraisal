package evepraisal

import (
	"sort"
	"github.com/evepraisal/go-evepraisal/typedb"
	"github.com/spf13/viper"
	"strconv"
	"fmt"
	"github.com/dustin/go-humanize"
)

const AsteroidCategoryID int64 = 25

const MineralGroupID int64 = 18
const MoonMaterialsGroupID int64 = 427
const IceProductGroupID int64 = 423

var BuybackGroups = []int64{MineralGroupID, MoonMaterialsGroupID, IceProductGroupID}

const CorpIP int64 = 98210135
const Corp0MD int64 = 728517421

var IPOrgCorporations = []int64{CorpIP, Corp0MD}

func (appraisal *Appraisal) BuybackOffer() float64 {
	buybackOffer := appraisal.Buyback.Totals.Buy
	if appraisal.BuybackCap > 0 {
		maxBuyback := appraisal.Original.Totals.Buy * appraisal.BuybackCap / 100
		if maxBuyback < buybackOffer {
			buybackOffer = maxBuyback
		}
	}
	return float64(int64(buybackOffer + 0.99));
}

func (appraisal *Appraisal) BuybackWarning() string {
	buybackMaxVolume := viper.GetFloat64("buyback-max-volume")
	if appraisal.Original.Totals.Volume > buybackMaxVolume && !appraisal.OnlyCompressedOre() {
		return fmt.Sprintf("NOTE: Since the volume is greater than %s m3 and contains more than Compressed Ore, it must be in H-T40Z to be accepted", humanize.Commaf(buybackMaxVolume))
	}
	return ""
}

func (appraisal *Appraisal) AverageBuybackPercentage() int64 {
	if appraisal.Original.Totals.Buy > 0 {
		return int64(appraisal.BuybackOffer() * 100 / appraisal.Original.Totals.Buy + 0.5)
	}
	return 0
}

func (appraisal *Appraisal) IsBuybackCapped() bool {
	return appraisal.BuybackOffer() < appraisal.Buyback.Totals.Buy
}

func (appraisal *Appraisal) BuybackReady() bool {
	if len(appraisal.Original.Items) == 0 {
		return false
	}

	for _, item := range appraisal.Original.Items {
		if item.Rejected {
			return false
		}
	}

	return true
}

func (app *App) DecoratedAdjustments() (adjustments map[string]string) {
	adjustments = make(map[string]string)
	for name, value := range viper.GetStringMapString("adjustments") {
		t, ok := app.TypeDB.GetType(name)
		if ok {
			adjustments[t.Name] = value
		}
	}
	return
}

func (app *App) buybackAdjustments() (adjustments Adjustments) {
	adjustments = make(map[int64]float64)
	for name, value := range viper.GetStringMapString("adjustments") {
		t, ok := app.TypeDB.GetType(name)
		adjustment, err := strconv.ParseFloat(value, 64)
		if ok && err == nil {
			adjustments[t.ID] = adjustment
		}
	}
	return
}

func (app *App) calculateBuyback(originalItems []AppraisalItem) (modifiedItems []AppraisalItem, buyback ItemsAndTotals) {
	adjustments := app.buybackAdjustments()

	buybackMap := make(map[string]*AppraisalItem)

	modifiedItems = make([]AppraisalItem, 0, len(originalItems))
	for _, item := range originalItems {
		if !item.Rejected {
			itemMap := make(map[string]*AppraisalItem)
			item.Qualifier, item.Efficiency = app.collectBuybackItems(itemMap, "DIRECT", 100, item.TypeID, item.Quantity)
			item.Buyback.Items = make([]AppraisalItem, 0, len(itemMap))
			for _, bbitem := range itemMap {
				item.Buyback.Items = append(item.Buyback.Items, *bbitem)
				app.updateBuybackItems(buybackMap, "DIRECT", 100, bbitem.Name, bbitem.TypeID, bbitem.Quantity)
			}
			sort.Sort(ByQuantity(item.Buyback.Items))
			app.priceAppraisalItems(item.Buyback.Items, &item.Buyback.Totals, "jita", adjustments)
		}
		modifiedItems = append(modifiedItems, item)
	}

	for _, bbitem := range buybackMap {
		buyback.Items = append(buyback.Items, *bbitem)
	}
	app.priceAppraisalItems(buyback.Items, &buyback.Totals, "jita", adjustments)
	sort.Sort(ByQuantity(buyback.Items))
	return
}

func (app *App) collectBuybackItems(itemMap map[string]*AppraisalItem, qualifier string, efficiency float64, typeID int64, quantity int64) (string, float64) {
	t, _ := app.TypeDB.GetTypeByID(typeID)

	portion := quantity / t.PortionSize
	if inBuybackGroup(t.GroupID) {
		app.updateBuybackItems(itemMap, qualifier, efficiency, t.Name, t.ID, portion)
	} else {
		if t.CategoryID == AsteroidCategoryID || t.GroupID == IceProductGroupID {
			qualifier, efficiency = "REFINE", 85
		} else {
			qualifier, efficiency = "REPROCESS", 55
		}
		for _, material := range t.Materials {
			app.collectBuybackItems(itemMap, qualifier, efficiency, material.TypeID, portion*material.Quantity)
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

func (app *App) ableToBuyback(t typedb.EveType) bool {
	if inBuybackGroup(t.GroupID) {
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

func inBuybackGroup(groupID int64) bool {
	for _, buybackGroupID := range BuybackGroups {
		if groupID == buybackGroupID {
			return true
		}
	}
	return false
}

type ByQuantity []AppraisalItem

func (a ByQuantity) Len() int           { return len(a) }
func (a ByQuantity) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByQuantity) Less(i, j int) bool { return a[i].Quantity > a[j].Quantity }