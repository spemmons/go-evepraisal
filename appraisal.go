package evepraisal

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/evepraisal/go-evepraisal/parsers"
	"github.com/evepraisal/go-evepraisal/typedb"
)

var (
	ErrNoValidLinesFound = fmt.Errorf("No valid lines found")
)

type Totals struct {
	Buy    float64 `json:"buy"`
	Sell   float64 `json:"sell"`
	Volume float64 `json:"volume"`
}

type ItemsAndTotals struct {
	Totals Totals          `json:"totals"`
	Items  []AppraisalItem `json:"items"`
}

type Appraisal struct {
	ID           string         `json:"id,omitempty"`
	Created      int64          `json:"created"`
	Kind         string         `json:"kind"`
	MarketName   string         `json:"market_name"`
	Original     ItemsAndTotals `json:"original"`
	Buyback      ItemsAndTotals `json:"buyback"`
	BuybackCap   float64        `json:"buyback_cap,omitempty"`
	Raw          string         `json:"raw"`
	Unparsed     map[int]string `json:"unparsed"`
	OwnerID      int64          `json:"owner_id,omitempty"`
	User         *User          `json:"user,omitempty"`
	Private      bool           `json:"private"`
	PrivateToken string         `json:"private_token,omitempty"`
	UserName     string         `json:"user_name,omitempty"`
}

func (appraisal *Appraisal) CreatedTime() time.Time {
	return time.Unix(appraisal.Created, 0)
}

func (appraisal *Appraisal) String() string {
	appraisalID := appraisal.ID
	if appraisalID == "" {
		appraisalID = "-"
	}
	s := fmt.Sprintf(
		"[Appraisal] id=%s, market=%s, kind=%s, items=%d, unparsed=%d",
		appraisalID, appraisal.MarketName, appraisal.Kind, len(appraisal.Original.Items), len(appraisal.Unparsed))
	if appraisal.User != nil {
		s += ", user=" + appraisal.User.CharacterName
	}
	if appraisal.Private {
		s += ", private"
	}
	return s
}

type AppraisalItem struct {
	Name       string  `json:"name"`
	TypeID     int64   `json:"typeID"`
	TypeName   string  `json:"typeName"`
	TypeVolume float64 `json:"typeVolume"`
	Quantity   int64   `json:"quantity"`
	Prices     Prices  `json:"prices"`
	Rejected   bool
	Qualifier  string
	Efficiency float64
	Adjustment float64 `json:"adjustment,omitempty"`
	Buyback    ItemsAndTotals
	Extra struct {
		Fitted     bool    `json:"fitted,omitempty"`
		Dropped    bool    `json:"dropped,omitempty"`
		Destroyed  bool    `json:"destroyed,omitempty"`
		Location   string  `json:"location,omitempty"`
		PlayerName string  `json:"player_name,omitempty"`
		Routed     bool    `json:"routed,omitempty"`
		Volume     float64 `json:"volume,omitempty"`
		Distance   string  `json:"distance,omitempty"`
		BPC        bool    `json:"bpc"`
		BPCRuns    int64   `json:"bpcRuns,omitempty"`
	} `json:"meta,omitempty"`
}

func (i AppraisalItem) DisplayName() string {
	if i.TypeName != "" {
		return i.TypeName
	} else {
		return i.Name
	}
}

func (i AppraisalItem) SellTotal() float64 {
	return float64(i.Quantity) * i.EffectiveAdjustment() * i.Prices.Sell.Min
}

func (i AppraisalItem) BuyTotal() float64 {
	return float64(i.Quantity) * i.EffectiveAdjustment() * i.Prices.Buy.Max
}

func (i AppraisalItem) SellISKVolume() float64 {
	return i.EffectiveAdjustment() * i.Prices.Sell.Min / i.TypeVolume
}

func (i AppraisalItem) BuyISKVolume() float64 {
	return i.EffectiveAdjustment() * i.Prices.Buy.Max / i.TypeVolume
}

func (i AppraisalItem) EffectiveAdjustment() float64 {
	if i.Adjustment == 0 {
		return 1
	}
	return i.Adjustment / 100.0
}

func (i AppraisalItem) SingleRepresentativePrice() float64 {
	if i.Prices.Sell.Min != 0 {
		return i.EffectiveAdjustment() * i.Prices.Sell.Min
	} else {
		return i.EffectiveAdjustment() * i.Prices.Buy.Max
	}
}

func (i AppraisalItem) RepresentativePrice() float64 {
	return float64(i.Quantity) * i.SingleRepresentativePrice()
}

type Prices struct {
	All      PriceStats `json:"all"`
	Buy      PriceStats `json:"buy"`
	Sell     PriceStats `json:"sell"`
	Updated  time.Time  `json:"updated"`
	Strategy string     `json:"strategy"`
	Basis    string     `json:"basis,omitempty"`
}

func (prices Prices) String() string {
	return fmt.Sprintf("Sell = %fISK, Buy = %fISK (Updated %s) (Using %s)", prices.Sell.Min, prices.Buy.Max, prices.Updated, prices.Strategy)
}

func (prices Prices) Set(price float64) Prices {
	prices.All.Average = price
	prices.All.Max = price
	prices.All.Min = price
	prices.All.Median = price
	prices.All.Percentile = price

	prices.Buy.Average = price
	prices.Buy.Max = price
	prices.Buy.Min = price
	prices.Buy.Median = price
	prices.Buy.Percentile = price

	prices.Sell.Average = price
	prices.Sell.Max = price
	prices.Sell.Min = price
	prices.Sell.Median = price
	prices.Sell.Percentile = price

	return prices
}

func (prices Prices) Add(p Prices) Prices {
	prices.All.Average += p.All.Average
	prices.All.Max += p.All.Max
	prices.All.Min += p.All.Min
	prices.All.Median += p.All.Median
	prices.All.Percentile += p.All.Percentile
	prices.All.Stddev += p.All.Stddev
	prices.All.Volume += p.All.Volume

	prices.Buy.Average += p.Buy.Average
	prices.Buy.Max += p.Buy.Max
	prices.Buy.Min += p.Buy.Min
	prices.Buy.Median += p.Buy.Median
	prices.Buy.Percentile += p.Buy.Percentile
	prices.Buy.Stddev += p.Buy.Stddev
	prices.Buy.Volume += p.Buy.Volume

	prices.Sell.Average += p.Sell.Average
	prices.Sell.Max += p.Sell.Max
	prices.Sell.Min += p.Sell.Min
	prices.Sell.Median += p.Sell.Median
	prices.Sell.Percentile += p.Sell.Percentile
	prices.Sell.Stddev += p.Sell.Stddev
	prices.Sell.Volume += p.Sell.Volume
	return prices
}

func (prices Prices) Sub(p Prices) Prices {
	prices.All.Average -= p.All.Average
	prices.All.Max -= p.All.Max
	prices.All.Min -= p.All.Min
	prices.All.Median -= p.All.Median
	prices.All.Percentile -= p.All.Percentile
	prices.All.Stddev -= p.All.Stddev
	prices.All.Volume += p.All.Volume

	prices.Buy.Average -= p.Buy.Average
	prices.Buy.Max -= p.Buy.Max
	prices.Buy.Min -= p.Buy.Min
	prices.Buy.Median -= p.Buy.Median
	prices.Buy.Percentile -= p.Buy.Percentile
	prices.Buy.Stddev -= p.Buy.Stddev
	prices.Buy.Volume += p.Buy.Volume

	prices.Sell.Average -= p.Sell.Average
	prices.Sell.Max -= p.Sell.Max
	prices.Sell.Min -= p.Sell.Min
	prices.Sell.Median -= p.Sell.Median
	prices.Sell.Percentile -= p.Sell.Percentile
	prices.Sell.Stddev -= p.Sell.Stddev
	prices.Sell.Volume += p.Sell.Volume
	return prices
}

func (prices Prices) Mul(quantity float64) Prices {
	prices.All.Average *= quantity
	prices.All.Max *= quantity
	prices.All.Min *= quantity
	prices.All.Median *= quantity
	prices.All.Percentile *= quantity
	prices.All.Stddev *= quantity

	prices.Buy.Average *= quantity
	prices.Buy.Max *= quantity
	prices.Buy.Min *= quantity
	prices.Buy.Median *= quantity
	prices.Buy.Percentile *= quantity
	prices.Buy.Stddev *= quantity

	prices.Sell.Average *= quantity
	prices.Sell.Max *= quantity
	prices.Sell.Min *= quantity
	prices.Sell.Median *= quantity
	prices.Sell.Percentile *= quantity
	prices.Sell.Stddev *= quantity
	return prices
}

type PriceStats struct {
	Average    float64 `json:"avg"`
	Max        float64 `json:"max"`
	Median     float64 `json:"median"`
	Min        float64 `json:"min"`
	Percentile float64 `json:"percentile"`
	Stddev     float64 `json:"stddev"`
	Volume     int64   `json:"volume"`
	OrderCount int64   `json:"order_count"`
}

type Adjustments map[int64]float64

var EmptyAdjustments = map[int64]float64{}

func (app *App) PricesForItem(market string, item AppraisalItem) (Prices, error) {
	if item.Extra.BPC {
		tName := strings.TrimSuffix(item.TypeName, " Blueprint")
		bpType, ok := app.TypeDB.GetType(tName)
		if !ok {
			log.Printf("WARN: parsed out name that isn't a type: %q", tName)
			return Prices{}, nil
		}

		marketMarket := market
		// If the user selected "universe" as the market then it is fairly likely that someone has a
		// rediculously low price in a station no one wants to travel to. To avoid negative "value"
		// for blueprint copies, we're forcing this item to be sold at jita prices Z
		if marketMarket == "universe" {
			marketMarket = "jita"
		}

		marketPrices := Prices{Strategy: "bpc"}
		for _, product := range bpType.BlueprintProducts {
			p, ok := app.PriceDB.GetPrice(marketMarket, product.TypeID)
			if !ok {
				log.Printf("WARN: No market data for type (%d %s)", item.TypeID, item.TypeName)
				continue
			}

			marketPrices = marketPrices.Add(p.Set(p.Sell.Min).Mul(float64(product.Quantity)))
		}

		manufacturedPrices := Prices{Strategy: "pbc"}
		for _, component := range bpType.Components {
			p, ok := app.PriceDB.GetPrice(market, component.TypeID)
			if !ok {
				log.Println("Failed getting getting price for component", component.TypeID)
				continue
			}
			manufacturedPrices = manufacturedPrices.Add(p.Set(math.Min(p.Sell.Min, p.Buy.Max)).Mul(float64(component.Quantity)))
		}

		// Assume Industry V (+10%) and misc costs (-1%)
		manufacturedPrices = manufacturedPrices.Mul(0.91)
		// prices := marketPrices.Sub(manufacturedPrices).Mul(float64(item.Extra.BPCRuns))

		log.Println("BPC Name: ", item.TypeName)
		log.Println("BPC materials:", manufacturedPrices)
		log.Println("BPC item value:", marketPrices)
		log.Println("BPC price (1 run):", marketPrices.Sub(manufacturedPrices))
		return Prices{}, nil
		// return prices, nil
	}

	return app.GetAdjustedPriceForItem(market, item), nil
}

type OreYields struct {
	Name   string
	Yields map[string]int64
}

var AllOreYields = []OreYields{
	{"Arkonor", map[string]int64{"Crimson": 5, "Prime": 10, "Flawless": 15}},
	{"Bistot", map[string]int64{"Triclinic": 5, "Monoclinic": 10, "Cubic": 15}},
	{"Crokite", map[string]int64{"Sharp": 5, "Crystalline": 10, "Pellucid": 15}},
	{"Dark", map[string]int64{"Onyx": 5, "Obsidian": 10, "Jet": 15}},
	{"Gneiss", map[string]int64{"Iridescent": 5, "Prismatic": 10, "Brilliant": 15}},
	{"Hedbergite", map[string]int64{"Vitric": 5, "Glazed": 10, "Lustrous": 15}},
	{"Hemorphite", map[string]int64{"Vivid": 5, "Radiant": 10, "Scintillating": 15}},
	{"Jaspet", map[string]int64{"Pure": 5, "Pristine": 10, "Immaculate": 15}},
	{"Kernite", map[string]int64{"Luminous": 5, "Fiery": 10, "Resplendant": 15}},
	{"Mercoxit", map[string]int64{"Magma": 5, "Vitreous": 10}},
	{"Omber", map[string]int64{"Silvery": 5, "Golden": 10, "Platinoid": 15}},
	{"Plagioclase", map[string]int64{"Azure": 5, "Rich": 10, "Sparkling": 15}},
	{"Pyroxeres", map[string]int64{"Solid": 5, "Viscous": 10, "Opulent": 15}},
	{"Scordite", map[string]int64{"Condensed": 5, "Massive": 10, "Glossy": 15}},
	{"Spodumain", map[string]int64{"Bright": 5, "Gleaming": 10, "Dazzling": 15}},
	{"Veldspar", map[string]int64{"Concentrated": 5, "Dense": 10, "Stable": 15}},
	{"Bitumens",map[string]int64{"Brimful": 15, "Glistening": 100}},
	{"Carnotite",map[string]int64{"Replete": 15, "Glowing": 100}},
	{"Chromite",map[string]int64{"Lavish": 15, "Shimmering": 100}},
	{"Cinnabar",map[string]int64{"Replete": 15, "Glowing": 100}},
	{"Cobaltite",map[string]int64{"Copious": 15, "Twinkling": 100}},
	{"Coesite",map[string]int64{"Brimful": 15, "Glistening": 100}},
	{"Euxenite",map[string]int64{"Copious": 15, "Twinkling": 100}},
	{"Loparite",map[string]int64{"Bountiful": 15, "Shining": 100}},
	{"Monazite",map[string]int64{"Bountiful": 15, "Shining": 100}},
	{"Otavite",map[string]int64{"Lavish": 15, "Shimmering": 100}},
	{"Pollucite",map[string]int64{"Replete": 15, "Glowing": 100}},
	{"Scheelite",map[string]int64{"Copious": 15, "Twinkling": 100}},
	{"Sperrylite",map[string]int64{"Lavish": 15, "Shimmering": 100}},
	{"Sylvite",map[string]int64{"Brimful": 15, "Glistening": 100}},
	{"Titanite",map[string]int64{"Copious": 15, "Twinkling": 100}},
	{"Vanadinite",map[string]int64{"Lavish": 15, "Shimmering": 100}},
	{"Xenotime",map[string]int64{"Bountiful": 15, "Shining": 100}},
	{"Ytterbite",map[string]int64{"Bountiful": 15, "Shining": 100}},
	{"Zeolites",map[string]int64{"Brimful": 15, "Glistening": 100}},
	{"Zircon",map[string]int64{"Replete": 15, "Glowing": 100}},	
}

func (app *App) GetAdjustedPriceForItem(market string, item AppraisalItem) (prices Prices) {
	var prefix string

	if strings.HasPrefix(item.Name, "Compressed") {
		prefix = "Compressed "
	}

	for _, oreYield := range AllOreYields {
		 if strings.HasSuffix(item.Name, oreYield.Name) {
			 for adjective, modifier := range oreYield.Yields {
				 if strings.Contains(item.Name, adjective) {
				 	t, ok := app.TypeDB.GetType(prefix + oreYield.Name)
				 	if ok {
						prices, _ = app.PriceDB.GetPrice(market, t.ID)
						prices = prices.Mul(1 + float64(modifier) / 100)
						prices.Basis = fmt.Sprintf("%s%s +%d%%", prefix, oreYield.Name, modifier)
						return
					}
				 }
			 }
		 }
	}

	prices, _ = app.PriceDB.GetPrice(market, item.TypeID)
	return
}

func (appraisal *Appraisal) OnlyCompressedOre() bool {
	for _, item := range appraisal.Original.Items {
		if !strings.HasPrefix(item.Name, "Compressed") { // NOTE: this might allow more than ore through...
			return false
		}
	}
	return true
}

func (app *App) StringToAppraisal(market string, s string) (*Appraisal, error) {
	appraisal := &Appraisal{
		Created: time.Now().Unix(),
		Raw:     s,
	}

	result, unparsed := app.Parser(parsers.StringToInput(s))

	appraisal.Unparsed = filterUnparsed(unparsed)

	kind, err := findKind(result)
	if err != nil {
		return appraisal, err
	}
	appraisal.Kind = kind
	appraisal.MarketName = market

	appraisal.Original.Items = parserResultToAppraisalItems(result)
	app.priceAppraisalItems(appraisal.Original.Items, &appraisal.Original.Totals, market, EmptyAdjustments)

	appraisal.Original.Items, appraisal.Buyback = app.calculateBuyback(appraisal.Original.Items)

	return appraisal, nil
}

func (app *App) priceAppraisalItems(items []AppraisalItem, totals *Totals, market string, adjustments map[int64]float64) {
	for i := 0; i < len(items); i++ {
		t, ok := app.TypeDB.GetType(items[i].Name)
		if !ok {
			log.Printf("WARN: parsed out name that isn't a type: %q", items[i].Name)
			continue
		}
		items[i].TypeID = t.ID
		items[i].TypeName = t.Name
		if t.PackagedVolume != 0.0 {
			items[i].TypeVolume = t.PackagedVolume
		} else {
			items[i].TypeVolume = t.Volume
		}

		items[i].Rejected = false // !app.ableToBuyback(t) -- vipeer wants to accept anything...
		if items[i].Rejected {
			continue
		}

		prices, err := app.PricesForItem(market, items[i])
		if err != nil {
			continue
		}

		items[i].Prices = prices

		baseAdjustment := adjustments[BaseAdjustmentID]
		if baseAdjustment != 0 {
			if adjustment, ok := adjustments[t.ID]; ok {
				items[i].Adjustment = adjustment
			} else {
				items[i].Adjustment = baseAdjustment
			}
		}

		totals.Buy += items[i].BuyTotal()
		totals.Sell += items[i].SellTotal()
		totals.Volume += items[i].TypeVolume * float64(items[i].Quantity)
	}
}

func findKind(result parsers.ParserResult) (string, error) {
	largestLines := -1
	largestLinesParser := "unknown"
	switch r := result.(type) {
	default:
		return largestLinesParser, fmt.Errorf("unexpected type %T", r)
	case *parsers.MultiParserResult:
		if len(r.Results) == 0 {
			return largestLinesParser, ErrNoValidLinesFound
		}
		for _, subResult := range r.Results {
			if len(subResult.Lines()) > largestLines {
				largestLines = len(subResult.Lines())
				largestLinesParser = subResult.Name()
			}
		}
	}
	return largestLinesParser, nil
}

func parserResultToAppraisalItems(result parsers.ParserResult) []AppraisalItem {
	var items []AppraisalItem
	switch r := result.(type) {
	default:
		log.Printf("unexpected type %T", r)
	case *parsers.MultiParserResult:
		for _, subResult := range r.Results {
			items = append(items, parserResultToAppraisalItems(subResult)...)
		}
	case *parsers.AssetList:
		for _, item := range r.Items {
			items = append(items, AppraisalItem{Name: item.Name, Quantity: item.Quantity})
		}
	case *parsers.CargoScan:
		for _, item := range r.Items {
			newItem := AppraisalItem{
				Name:     item.Name,
				Quantity: item.Quantity,
			}
			newItem.Extra.BPC = item.BPC
			if item.BPC {
				newItem.Extra.BPCRuns = 1
			}
			items = append(items, newItem)
		}
	case *parsers.Contract:
		for _, item := range r.Items {
			newItem := AppraisalItem{
				Name:     item.Name,
				Quantity: item.Quantity,
			}
			newItem.Extra.Fitted = item.Fitted
			newItem.Extra.BPC = item.BPC
			newItem.Extra.BPCRuns = item.BPCRuns
			items = append(items, newItem)
		}
	case *parsers.DScan:
		for _, item := range r.Items {
			items = append(items, AppraisalItem{Name: item.Name, Quantity: 1})
		}
	case *parsers.EFT:
		items = append(items, AppraisalItem{Name: r.Ship, Quantity: 1})
		for _, item := range r.Items {
			items = append(items, AppraisalItem{Name: item.Name, Quantity: item.Quantity})
		}
	case *parsers.Fitting:
		for _, item := range r.Items {
			items = append(items, AppraisalItem{Name: item.Name, Quantity: item.Quantity})
		}
	case *parsers.Industry:
		for _, item := range r.Items {
			items = append(items, AppraisalItem{Name: item.Name, Quantity: item.Quantity})
		}
	case *parsers.Killmail:
		for _, item := range r.Dropped {
			newItem := AppraisalItem{
				Name:     item.Name,
				Quantity: item.Quantity,
			}
			newItem.Extra.Dropped = true
			newItem.Extra.Location = item.Location
			items = append(items, newItem)
		}
		for _, item := range r.Destroyed {
			newItem := AppraisalItem{
				Name:     item.Name,
				Quantity: item.Quantity,
			}
			newItem.Extra.Destroyed = true
			newItem.Extra.Location = item.Location
			items = append(items, newItem)
		}
	case *parsers.Listing:
		for _, item := range r.Items {
			items = append(items, AppraisalItem{Name: item.Name, Quantity: item.Quantity})
		}
	case *parsers.LootHistory:
		for _, item := range r.Items {
			newItem := AppraisalItem{
				Name:     item.Name,
				Quantity: item.Quantity,
			}
			newItem.Extra.PlayerName = item.PlayerName
			items = append(items, newItem)
		}
	case *parsers.PI:
		for _, item := range r.Items {
			newItem := AppraisalItem{
				Name:     item.Name,
				Quantity: item.Quantity,
			}
			newItem.Extra.Routed = item.Routed
			newItem.Extra.Volume = item.Volume
			items = append(items, newItem)
		}
	case *parsers.SurveyScan:
		for _, item := range r.Items {
			newItem := AppraisalItem{
				Name:     item.Name,
				Quantity: item.Quantity,
			}
			newItem.Extra.Distance = item.Distance
			items = append(items, newItem)
		}
	case *parsers.ViewContents:
		for _, item := range r.Items {
			newItem := AppraisalItem{
				Name:     item.Name,
				Quantity: item.Quantity,
			}
			newItem.Extra.Location = item.Location
			items = append(items, newItem)
		}
	case *parsers.Wallet:
		for _, item := range r.ItemizedTransactions {
			items = append(items,
				AppraisalItem{
					Name:     item.Name,
					Quantity: item.Quantity,
				})
		}
	case *parsers.HeuristicResult:
		for _, item := range r.Items {
			items = append(items, AppraisalItem{Name: item.Name, Quantity: item.Quantity})
		}
	}

	itemMap := make(map[string]AppraisalItem)
	quantityMap := make(map[string]int64)
	for _, item := range items {
		item.Name = strings.Trim(item.Name, " \t")
		key := strings.ToUpper(item.Name)
		itemMap[key] = item
		quantityMap[key] += item.Quantity
	}

	returnItems := make([]AppraisalItem, 0, len(itemMap))
	for key, item := range itemMap {
		item.Quantity = quantityMap[key]
		returnItems = append(returnItems, item)
	}

	return returnItems
}

func filterUnparsed(unparsed map[int]string) map[int]string {
	for lineNum, line := range unparsed {
		if strings.Trim(line, " \t") == "" {
			delete(unparsed, lineNum)
		}
	}
	return unparsed
}

func priceByComponents(t typedb.EveType, priceDB PriceDB, market string) Prices {
	var prices Prices
	for _, component := range t.Components {
		p, ok := priceDB.GetPrice(market, component.TypeID)
		if !ok {
			continue
		}
		prices = prices.Add(p.Mul(float64(component.Quantity)))
	}
	return prices
}
