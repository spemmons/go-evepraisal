package staticdump

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/evepraisal/go-evepraisal/typedb"
	"github.com/sethgrid/pester"
)

var userAgent = "go-evepraisal"

// FindLastStaticDumpURL returns the URL of the last eve static data dump
func FindLastStaticDumpURL(client *pester.Client) (string, error) {
	i := 0
	current := time.Now()
	for i < 200 {
		url := "https://cdn1.eveonline.com/data/sde/tranquility/sde-" + current.Format("20060102") + "-TRANQUILITY.zip"
		req, err := http.NewRequest("HEAD", url, nil)
		if err != nil {
			return "", err
		}
		req.Header.Add("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}

		switch resp.StatusCode {
		case 200, 304:
			return url, nil
		case 404:
			current = current.Add(-24 * time.Hour)
			continue
		default:
			return "", fmt.Errorf("Unexpected response when trying to find last static dump: %s", resp.Status)
		}
	}
	return "", errors.New("Could not find latest static dump URL")
}

func downloadTypes(client *pester.Client, staticDumpURL string, staticDataPath string) error {
	out, err := os.Create(staticDataPath)
	defer out.Close()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", staticDumpURL, nil)
	if err != nil {
		return err
	}

	req.Header.Add("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	log.Printf("Successfully wrote %d bytes to %s", n, staticDataPath)
	return nil
}

// Type is an eve online type
type Type struct {
	GroupID       int64 `yaml:"groupID"`
	MarketGroupID int64 `yaml:"marketGroupID"`
	Name          struct {
		En string
	}
	Published bool
	Volume    float64
	BasePrice float64
}

// Blueprint is an eve online blueprint
type Blueprint struct {
	BlueprintTypeID int64 `yaml:"blueprintTypeID"`
	Activities      struct {
		Manufacturing struct {
			Materials []struct {
				Quantity int64
				TypeID   int64 `yaml:"typeID"`
			}
			Products []struct {
				Quantity int64
				TypeID   int64 `yaml:"typeID"`
			}
		}
	}
}

func loadtypes(staticDataPath string) ([]typedb.EveType, error) {
	r, err := zip.OpenReader(staticDataPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var allTypes map[int64]Type
	err = loadDataFromZipFile(r, "sde/fsd/typeIDs.yaml", &allTypes)
	if err != nil {
		return nil, err
	}
	log.Printf("Loaded %d types", len(allTypes))

	var allBlueprints map[int64]Blueprint
	err = loadDataFromZipFile(r, "sde/fsd/blueprints.yaml", &allBlueprints)
	if err != nil {
		return nil, err
	}
	log.Printf("Loaded %d blueprints", len(allBlueprints))

	blueprintsByProductType := make(map[int64][]Blueprint)
	for _, blueprint := range allBlueprints {
		for _, product := range blueprint.Activities.Manufacturing.Products {
			blueprints, ok := blueprintsByProductType[product.TypeID]
			if ok {
				blueprintsByProductType[product.TypeID] = append(blueprints, blueprint)
			} else {
				blueprintsByProductType[product.TypeID] = []Blueprint{blueprint}
			}
		}
	}

	types := make([]typedb.EveType, 0)
	for typeID, t := range allTypes {

		// Yep, there is at least one type that isn't named.
		if t.Name.En == "" {
			continue
		}

		eveType := typedb.EveType{
			ID:                typeID,
			GroupID:           t.GroupID,
			MarketGroupID:     t.MarketGroupID,
			Name:              t.Name.En,
			Volume:            t.Volume,
			BasePrice:         t.BasePrice,
			BlueprintProducts: resolveBlueprintProducts(blueprintsByProductType, typeID),
			Components:        resolveComponents(blueprintsByProductType, typeID),
			BaseComponents:    flattenComponents(resolveBaseComponents(blueprintsByProductType, typeID, 1, 5)),
		}
		types = append(types, eveType)
	}

	return types, nil
}

func resolveBlueprintProducts(blueprintsByProductType map[int64][]Blueprint, typeID int64) []typedb.Component {
	blueprints, ok := blueprintsByProductType[typeID]
	if !ok || len(blueprints) == 0 {
		return nil
	}

	bp := blueprints[0]
	var components []typedb.Component
	for _, material := range bp.Activities.Manufacturing.Products {
		components = append(components, typedb.Component{Quantity: material.Quantity, TypeID: material.TypeID})
	}
	return components
}

func resolveComponents(blueprintsByProductType map[int64][]Blueprint, typeID int64) []typedb.Component {
	blueprints, ok := blueprintsByProductType[typeID]
	if !ok || len(blueprints) == 0 {
		return nil
	}

	bp := blueprints[0]
	var components []typedb.Component
	for _, material := range bp.Activities.Manufacturing.Materials {
		components = append(components, typedb.Component{Quantity: material.Quantity, TypeID: material.TypeID})
	}
	return components
}

func flattenComponents(components []typedb.Component) []typedb.Component {
	m := make(map[typedb.Component]int64)
	for _, component := range components {
		qty := component.Quantity
		component.Quantity = 0
		m[component] += qty
	}

	s := make([]typedb.Component, 0, len(m))
	for component, qty := range m {
		component.Quantity = qty
		s = append(s, component)
	}
	return s
}

func resolveBaseComponents(blueprintsByProductType map[int64][]Blueprint, typeID int64, multiplier int64, left int) []typedb.Component {
	if left == 0 {
		return nil
	}

	blueprints, ok := blueprintsByProductType[typeID]
	if !ok || len(blueprints) == 0 {
		return nil
	}

	bp := blueprints[0]
	var components []typedb.Component
	for _, material := range bp.Activities.Manufacturing.Materials {
		r := resolveBaseComponents(blueprintsByProductType, material.TypeID, material.Quantity*multiplier, left-1)
		if r == nil {
			components = append(components, typedb.Component{Quantity: material.Quantity * multiplier, TypeID: material.TypeID})
		} else {
			components = append(components, r...)
		}
	}
	return components
}
