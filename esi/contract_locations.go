package esi

import "fmt"

type SystemSov struct {
	SystemID      int64 `json:"system_id"`                //system_id (integer): system_id integer ,
	AllianceID    int64 `json:"alliance_id,omitempty"`    //alliance_id (integer, optional): alliance_id integer ,
	CorporationID int64 `json:"corporation_id,omitempty"` //corporation_id (integer, optional): corporation_id integer ,
	FactionID     int64 `json:"faction_id,omitempty"`     //faction_id (integer, optional): faction_id integer
}

type UniverseConstellation struct {
	ConstellationID int64 `json:"constellation_id"` //constellation_id (integer): The constellation this solar system is in ,
	Name            string                          //name (string): The full name of the structure ,
	RegionID        int64 `json:"region_id"`        //region_id (integer): The solar region this station is in,
}

type UniverseSystem struct {
	SystemID        int64 `json:"system_id"`        //system_id (integer): The solar system this station is in,
	Name            string                          //name (string): The full name of the structure ,
	ConstellationID int64 `json:"constellation_id"` //constellation_id (integer): The constellation this solar system is in ,
}

type UniverseStation struct {
	StationID int64                          //station_id (integer): station_id integer ,
	Name      string                         //name (string): The full name of the structure ,
	OwnerID   int64 `json:"owner,omitempty"` //owner (integer, optional): ID of the corporation that controls this station ,
	SystemID  int64 `json:"system_id"`       //system_id (integer): The solar system this station is in,
	TypeID    int64 `json:"type_id"`         //type_id (integer, optional): type_id integer ,
}

type UniverseStructure struct {
	Name     string                           //name (string): The full name of the structure ,
	SystemID int64 `json:"solar_system_id"`   //solar_system_id (integer): Solar system in which the structure is located. ,
	TypeID   int64 `json:"type_id,omitempty"` //type_id (integer, optional): type_id integer ,
}

func (cf *ContractFetcher) FindLocation(locationID int64) (int64, string, bool) {
	fmt.Printf("LOCATION: %v\n", locationID)

	station, found := cf.getUniverseStation(locationID)
	if found {
		return station.SystemID, station.Name, true
	}

	structure, found := cf.getUniverseStructure(locationID)
	if found {
		return structure.SystemID, structure.Name, true
	}

	return 0, "", false
}

func (cf *ContractFetcher) FindRegionForSystemID(systemID int64) (int64, bool) {
	system, found := cf.getUniverseSystem(systemID)
	if !found {
		return 0, false
	}

	constellation, found := cf.getUniverseConstellation(system.ConstellationID)
	if !found {
		return 0, false
	}

	return constellation.RegionID, constellation.RegionID != 0
}

var constellationMap = map[int64]*UniverseConstellation{}

func (cf *ContractFetcher) getUniverseConstellation(constellationID int64) (*UniverseConstellation, bool) {
	var constellation *UniverseConstellation
	constellation, found := constellationMap[constellationID]
	if !found {
		constellation = new(UniverseConstellation)
		url := fmt.Sprintf("%s/universe/constellations/%d/", cf.baseURL, constellationID)
		_ = fetchURL(cf.client, url, &constellation)
		constellationMap[constellationID] = constellation
	}
	return constellation, constellation.ConstellationID != 0
}

var systemMap = map[int64]*UniverseSystem{}

func (cf *ContractFetcher) getUniverseSystem(systemID int64) (*UniverseSystem, bool) {
	var system *UniverseSystem
	system, found := systemMap[systemID]
	if !found {
		system = new(UniverseSystem)
		url := fmt.Sprintf("%s/universe/systems/%d/", cf.baseURL, systemID)
		_ = fetchURL(cf.client, url, &system)
		systemMap[systemID] = system
	}
	return system, system.SystemID != 0
}

var stationMap = map[int64]*UniverseStation{}

func (cf *ContractFetcher) getUniverseStation(stationID int64) (*UniverseStation, bool) {
	var station *UniverseStation
	station, found := stationMap[stationID]
	if !found {
		station = new(UniverseStation)
		url := fmt.Sprintf("%s/universe/stations/%d/", cf.baseURL, stationID)
		_ = fetchURL(cf.client, url, &station)
		stationMap[stationID] = station
	}
	fmt.Printf("STATION: %+v\n", station)
	return station, station.SystemID != 0
}

func (cf *ContractFetcher) getUniverseStructure(locationID int64) (*UniverseStructure, bool) {
	result := new(UniverseStructure)
	url := fmt.Sprintf("%s/universe/structures/%d/?token=%s", cf.baseURL, locationID, cf.token.AccessToken)
	err := fetchURL(cf.client, url, result)
	return result, err == nil
}

var sovMap = map[int64]SystemSov{}

func (cf *ContractFetcher) FindAllianceForSystemID(systemID int64) (int64, bool) {
	if len(sovMap) == 0 {
		systems := make([]SystemSov, 0)
		url := fmt.Sprintf("%s/sovereignty/map/", cf.baseURL)
		err := fetchURL(cf.client, url, &systems)
		if err != nil {
			fmt.Printf("SOV MAP ERR: %v\n", err)
		} else {
			for _, system := range systems {
				sovMap[system.SystemID] = system
			}
		}
	}

	system, found := sovMap[systemID]
	if found {
		return system.AllianceID, true
	}

	return 0, false
}
