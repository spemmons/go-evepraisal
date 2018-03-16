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

func (of *OauthFetcher) FindLocation(locationID int64) (int64, string, bool) {
	station, found := of.getUniverseStation(locationID)
	if found {
		return station.SystemID, station.Name, true
	}

	structure, found := of.getUniverseStructure(locationID)
	if found {
		return structure.SystemID, structure.Name, true
	}

	fmt.Printf("UNEXPECTED ERROR - Unable to find location: %d\n", locationID)
	return 0, "", false
}

func (of *OauthFetcher) FindRegionForSystemID(systemID int64) (int64, bool) {
	system, found := of.getUniverseSystem(systemID)
	if !found {
		fmt.Printf("UNEXPECTED ERROR - System not found: %d\n", systemID)
		return 0, false
	}

	constellation, found := of.getUniverseConstellation(system.ConstellationID)
	if !found {
		fmt.Printf("UNEXPECTED ERROR - System %s constellation not found: %d\n", system.Name, system.ConstellationID)
		return 0, false
	}

	if constellation.RegionID == 0 {
		fmt.Printf("UNEXPECTED ERROR - Constellation %s region not found: %d\n", constellation.Name, system.ConstellationID)
		return 0, false
	}

	return constellation.RegionID, true
}

var constellationMap = map[int64]*UniverseConstellation{}

func (of *OauthFetcher) getUniverseConstellation(constellationID int64) (*UniverseConstellation, bool) {
	var constellation *UniverseConstellation
	constellation, found := constellationMap[constellationID]
	if !found {
		constellation = new(UniverseConstellation)
		url := fmt.Sprintf("%s/universe/constellations/%d/", of.baseURL, constellationID)
		_ = fetchURL(of.client, url, &constellation)
		constellationMap[constellationID] = constellation
	}
	return constellation, constellation.ConstellationID != 0
}

var systemMap = map[int64]*UniverseSystem{}

func (of *OauthFetcher) getUniverseSystem(systemID int64) (*UniverseSystem, bool) {
	var system *UniverseSystem
	system, found := systemMap[systemID]
	if !found {
		system = new(UniverseSystem)
		url := fmt.Sprintf("%s/universe/systems/%d/", of.baseURL, systemID)
		_ = fetchURL(of.client, url, &system)
		systemMap[systemID] = system
	}
	return system, system.SystemID != 0
}

var stationMap = map[int64]*UniverseStation{}

func (of *OauthFetcher) getUniverseStation(stationID int64) (*UniverseStation, bool) {
	var station *UniverseStation
	station, found := stationMap[stationID]
	if !found {
		station = new(UniverseStation)
		url := fmt.Sprintf("%s/universe/stations/%d/", of.baseURL, stationID)
		_ = fetchURL(of.client, url, &station)
		stationMap[stationID] = station
	}
	return station, station.SystemID != 0
}

func (of *OauthFetcher) getUniverseStructure(locationID int64) (*UniverseStructure, bool) {
	result := new(UniverseStructure)
	url := fmt.Sprintf("%s/universe/structures/%d/", of.baseURL, locationID)
	err := fetchURL(of.client, url, result)
	return result, err == nil
}

var sovMap = map[int64]SystemSov{}

func (of *OauthFetcher) FindAllianceForSystemID(systemID int64) (int64, bool) {
	if len(sovMap) == 0 {
		systems := make([]SystemSov, 0)
		url := fmt.Sprintf("%s/sovereignty/map/", of.baseURL)
		err := fetchURL(of.client, url, &systems)
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
