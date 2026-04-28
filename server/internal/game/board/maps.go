package board

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lukev/tm_server/internal/models"
)

type MapID string

const (
	MapBase        MapID = "base"
	MapArchipelago MapID = "archipelago"
	MapCustom      MapID = "custom"
	MapFireAndIce  MapID = "fire-and-ice"
	MapFjords      MapID = "fjords"
	MapLakes       MapID = "lakes"
	MapRevisedBase MapID = "revised-base"
)

type MapInfo struct {
	ID   MapID  `json:"id"`
	Name string `json:"name"`
}

type CustomMapDefinition struct {
	Name            string                 `json:"name,omitempty"`
	RowCount        int                    `json:"rowCount"`
	FirstRowColumns int                    `json:"firstRowColumns"`
	FirstRowLonger  bool                   `json:"firstRowLonger"`
	Rows            [][]models.TerrainType `json:"rows"`
}

type rowDefinition struct {
	R        int
	StartQ   int
	Terrains []models.TerrainType
}

type mapDefinition struct {
	Info MapInfo
	Rows []rowDefinition
}

type coordinateIndex struct {
	displayByHex   map[Hex]string
	hexByDisplayID map[string]Hex
}

var customMapInfo = MapInfo{ID: MapCustom, Name: "Custom"}

func (d mapDefinition) Layout() map[Hex]models.TerrainType {
	layout := make(map[Hex]models.TerrainType)
	for _, row := range d.Rows {
		for offset, terrain := range row.Terrains {
			layout[NewHex(row.StartQ+offset, row.R)] = terrain
		}
	}
	return layout
}

func baseRow(r int, terrains ...models.TerrainType) rowDefinition {
	return rowDefinition{
		R:        r,
		StartQ:   mapStartQForRow(r),
		Terrains: append([]models.TerrainType(nil), terrains...),
	}
}

func lakesRow(r int, terrains ...models.TerrainType) rowDefinition {
	return rowDefinition{
		R:        r,
		StartQ:   lakesStartQForRow(r),
		Terrains: append([]models.TerrainType(nil), terrains...),
	}
}

func fireAndIceRow(r int, terrains ...models.TerrainType) rowDefinition {
	return rowDefinition{
		R:        r,
		StartQ:   lakesStartQForRow(r),
		Terrains: append([]models.TerrainType(nil), terrains...),
	}
}

func mapStartQForRow(r int) int {
	if r%2 == 0 {
		return -(r / 2)
	}
	return -((r - 1) / 2)
}

func lakesStartQForRow(r int) int {
	if r%2 == 0 {
		return -(r / 2)
	}
	return -((r + 1) / 2)
}

func startQForRow(firstRowLonger bool, r int) int {
	if firstRowLonger {
		return mapStartQForRow(r)
	}
	return lakesStartQForRow(r)
}

func cloneRows(rows []rowDefinition) []rowDefinition {
	cloned := make([]rowDefinition, len(rows))
	for i, row := range rows {
		cloned[i] = rowDefinition{
			R:        row.R,
			StartQ:   row.StartQ,
			Terrains: append([]models.TerrainType(nil), row.Terrains...),
		}
	}
	return cloned
}

var baseRows = []rowDefinition{
	baseRow(0,
		models.TerrainPlains, models.TerrainMountain, models.TerrainForest, models.TerrainLake,
		models.TerrainDesert, models.TerrainWasteland, models.TerrainPlains, models.TerrainSwamp,
		models.TerrainWasteland, models.TerrainForest, models.TerrainLake, models.TerrainWasteland,
		models.TerrainSwamp,
	),
	baseRow(1,
		models.TerrainDesert, models.TerrainRiver, models.TerrainRiver, models.TerrainPlains,
		models.TerrainSwamp, models.TerrainRiver, models.TerrainRiver, models.TerrainDesert,
		models.TerrainSwamp, models.TerrainRiver, models.TerrainRiver, models.TerrainDesert,
	),
	baseRow(2,
		models.TerrainRiver, models.TerrainRiver, models.TerrainSwamp, models.TerrainRiver,
		models.TerrainMountain, models.TerrainRiver, models.TerrainForest, models.TerrainRiver,
		models.TerrainForest, models.TerrainRiver, models.TerrainMountain, models.TerrainRiver,
		models.TerrainRiver,
	),
	baseRow(3,
		models.TerrainForest, models.TerrainLake, models.TerrainDesert, models.TerrainRiver,
		models.TerrainRiver, models.TerrainWasteland, models.TerrainLake, models.TerrainRiver,
		models.TerrainWasteland, models.TerrainRiver, models.TerrainWasteland, models.TerrainPlains,
	),
	baseRow(4,
		models.TerrainSwamp, models.TerrainPlains, models.TerrainWasteland, models.TerrainLake,
		models.TerrainSwamp, models.TerrainPlains, models.TerrainMountain, models.TerrainDesert,
		models.TerrainRiver, models.TerrainRiver, models.TerrainForest, models.TerrainSwamp,
		models.TerrainLake,
	),
	baseRow(5,
		models.TerrainMountain, models.TerrainForest, models.TerrainRiver, models.TerrainRiver,
		models.TerrainDesert, models.TerrainForest, models.TerrainRiver, models.TerrainRiver,
		models.TerrainRiver, models.TerrainPlains, models.TerrainMountain, models.TerrainPlains,
	),
	baseRow(6,
		models.TerrainRiver, models.TerrainRiver, models.TerrainRiver, models.TerrainMountain,
		models.TerrainRiver, models.TerrainWasteland, models.TerrainRiver, models.TerrainForest,
		models.TerrainRiver, models.TerrainDesert, models.TerrainSwamp, models.TerrainLake,
		models.TerrainDesert,
	),
	baseRow(7,
		models.TerrainDesert, models.TerrainLake, models.TerrainPlains, models.TerrainRiver,
		models.TerrainRiver, models.TerrainRiver, models.TerrainLake, models.TerrainSwamp,
		models.TerrainRiver, models.TerrainMountain, models.TerrainPlains, models.TerrainMountain,
	),
	baseRow(8,
		models.TerrainWasteland, models.TerrainSwamp, models.TerrainMountain, models.TerrainLake,
		models.TerrainWasteland, models.TerrainForest, models.TerrainDesert, models.TerrainPlains,
		models.TerrainMountain, models.TerrainRiver, models.TerrainLake, models.TerrainForest,
		models.TerrainWasteland,
	),
}

var revisedBaseRows = []rowDefinition{
	baseRow(0,
		models.TerrainPlains, models.TerrainMountain, models.TerrainForest, models.TerrainLake,
		models.TerrainPlains, models.TerrainWasteland, models.TerrainPlains, models.TerrainSwamp,
		models.TerrainWasteland, models.TerrainLake, models.TerrainForest, models.TerrainWasteland,
		models.TerrainSwamp,
	),
	baseRow(1,
		models.TerrainDesert, models.TerrainRiver, models.TerrainRiver, models.TerrainDesert,
		models.TerrainSwamp, models.TerrainRiver, models.TerrainRiver, models.TerrainDesert,
		models.TerrainForest, models.TerrainRiver, models.TerrainRiver, models.TerrainDesert,
	),
	baseRow(2,
		models.TerrainRiver, models.TerrainRiver, models.TerrainSwamp, models.TerrainRiver,
		models.TerrainMountain, models.TerrainRiver, models.TerrainForest, models.TerrainRiver,
		models.TerrainSwamp, models.TerrainRiver, models.TerrainWasteland, models.TerrainRiver,
		models.TerrainRiver,
	),
	baseRow(3,
		models.TerrainForest, models.TerrainLake, models.TerrainDesert, models.TerrainRiver,
		models.TerrainRiver, models.TerrainWasteland, models.TerrainLake, models.TerrainRiver,
		models.TerrainWasteland, models.TerrainRiver, models.TerrainMountain, models.TerrainPlains,
	),
	baseRow(4,
		models.TerrainSwamp, models.TerrainPlains, models.TerrainWasteland, models.TerrainLake,
		models.TerrainDesert, models.TerrainPlains, models.TerrainForest, models.TerrainDesert,
		models.TerrainRiver, models.TerrainRiver, models.TerrainForest, models.TerrainSwamp,
		models.TerrainWasteland,
	),
	baseRow(5,
		models.TerrainMountain, models.TerrainForest, models.TerrainRiver, models.TerrainRiver,
		models.TerrainSwamp, models.TerrainMountain, models.TerrainRiver, models.TerrainRiver,
		models.TerrainRiver, models.TerrainPlains, models.TerrainMountain, models.TerrainDesert,
	),
	baseRow(6,
		models.TerrainRiver, models.TerrainRiver, models.TerrainRiver, models.TerrainMountain,
		models.TerrainRiver, models.TerrainWasteland, models.TerrainRiver, models.TerrainForest,
		models.TerrainRiver, models.TerrainDesert, models.TerrainSwamp, models.TerrainLake,
		models.TerrainPlains,
	),
	baseRow(7,
		models.TerrainDesert, models.TerrainLake, models.TerrainPlains, models.TerrainRiver,
		models.TerrainRiver, models.TerrainRiver, models.TerrainLake, models.TerrainSwamp,
		models.TerrainRiver, models.TerrainMountain, models.TerrainPlains, models.TerrainWasteland,
	),
	baseRow(8,
		models.TerrainLake, models.TerrainSwamp, models.TerrainMountain, models.TerrainLake,
		models.TerrainWasteland, models.TerrainForest, models.TerrainDesert, models.TerrainPlains,
		models.TerrainMountain, models.TerrainRiver, models.TerrainLake, models.TerrainForest,
		models.TerrainMountain,
	),
}

var lakesRows = []rowDefinition{
	lakesRow(0,
		models.TerrainMountain, models.TerrainLake, models.TerrainWasteland, models.TerrainPlains,
		models.TerrainDesert, models.TerrainLake, models.TerrainDesert, models.TerrainWasteland,
		models.TerrainRiver, models.TerrainRiver, models.TerrainForest, models.TerrainLake,
	),
	lakesRow(1,
		models.TerrainDesert, models.TerrainSwamp, models.TerrainForest, models.TerrainRiver,
		models.TerrainRiver, models.TerrainSwamp, models.TerrainPlains, models.TerrainRiver,
		models.TerrainForest, models.TerrainMountain, models.TerrainRiver, models.TerrainPlains,
		models.TerrainSwamp,
	),
	lakesRow(2,
		models.TerrainPlains, models.TerrainRiver, models.TerrainRiver, models.TerrainForest,
		models.TerrainWasteland, models.TerrainMountain, models.TerrainRiver, models.TerrainSwamp,
		models.TerrainLake, models.TerrainWasteland, models.TerrainRiver,
		models.TerrainDesert,
	),
	lakesRow(3,
		models.TerrainLake, models.TerrainWasteland, models.TerrainMountain, models.TerrainRiver,
		models.TerrainDesert, models.TerrainPlains, models.TerrainForest, models.TerrainRiver,
		models.TerrainRiver, models.TerrainDesert, models.TerrainRiver, models.TerrainSwamp,
		models.TerrainWasteland,
	),
	lakesRow(4,
		models.TerrainForest, models.TerrainDesert, models.TerrainRiver, models.TerrainSwamp,
		models.TerrainLake, models.TerrainRiver, models.TerrainRiver, models.TerrainWasteland,
		models.TerrainRiver, models.TerrainMountain, models.TerrainForest, models.TerrainPlains,
	),
	lakesRow(5,
		models.TerrainMountain, models.TerrainRiver, models.TerrainPlains, models.TerrainMountain,
		models.TerrainRiver, models.TerrainDesert, models.TerrainRiver, models.TerrainMountain,
		models.TerrainRiver, models.TerrainPlains, models.TerrainSwamp, models.TerrainLake,
		models.TerrainWasteland,
	),
	lakesRow(6,
		models.TerrainWasteland, models.TerrainRiver, models.TerrainRiver, models.TerrainRiver,
		models.TerrainWasteland, models.TerrainForest, models.TerrainPlains, models.TerrainSwamp,
		models.TerrainDesert, models.TerrainRiver, models.TerrainRiver, models.TerrainMountain,
	),
	lakesRow(7,
		models.TerrainDesert, models.TerrainLake, models.TerrainSwamp, models.TerrainRiver,
		models.TerrainLake, models.TerrainMountain, models.TerrainLake, models.TerrainRiver,
		models.TerrainRiver, models.TerrainMountain, models.TerrainForest, models.TerrainRiver,
		models.TerrainLake,
	),
	lakesRow(8,
		models.TerrainSwamp, models.TerrainPlains, models.TerrainRiver, models.TerrainForest,
		models.TerrainRiver, models.TerrainRiver, models.TerrainRiver, models.TerrainForest,
		models.TerrainWasteland, models.TerrainPlains, models.TerrainDesert, models.TerrainSwamp,
	),
}

var fireAndIceRows = []rowDefinition{
	fireAndIceRow(0,
		models.TerrainPlains, models.TerrainRiver, models.TerrainPlains, models.TerrainSwamp,
		models.TerrainDesert, models.TerrainRiver, models.TerrainMountain, models.TerrainForest,
		models.TerrainWasteland, models.TerrainLake, models.TerrainDesert, models.TerrainLake,
	),
	fireAndIceRow(1,
		models.TerrainWasteland, models.TerrainDesert, models.TerrainRiver, models.TerrainLake,
		models.TerrainMountain, models.TerrainWasteland, models.TerrainRiver, models.TerrainRiver,
		models.TerrainRiver, models.TerrainDesert, models.TerrainPlains, models.TerrainSwamp,
		models.TerrainMountain,
	),
	fireAndIceRow(2,
		models.TerrainForest, models.TerrainSwamp, models.TerrainRiver, models.TerrainRiver,
		models.TerrainRiver, models.TerrainPlains, models.TerrainForest, models.TerrainDesert,
		models.TerrainRiver, models.TerrainRiver, models.TerrainRiver, models.TerrainRiver,
	),
	fireAndIceRow(3,
		models.TerrainDesert, models.TerrainMountain, models.TerrainForest, models.TerrainDesert,
		models.TerrainSwamp, models.TerrainRiver, models.TerrainLake, models.TerrainWasteland,
		models.TerrainPlains, models.TerrainRiver, models.TerrainForest, models.TerrainLake,
		models.TerrainForest,
	),
	fireAndIceRow(4,
		models.TerrainRiver, models.TerrainRiver, models.TerrainPlains, models.TerrainRiver,
		models.TerrainRiver, models.TerrainWasteland, models.TerrainSwamp, models.TerrainForest,
		models.TerrainMountain, models.TerrainRiver, models.TerrainPlains, models.TerrainSwamp,
	),
	fireAndIceRow(5,
		models.TerrainForest, models.TerrainWasteland, models.TerrainRiver, models.TerrainRiver,
		models.TerrainForest, models.TerrainRiver, models.TerrainRiver, models.TerrainRiver,
		models.TerrainPlains, models.TerrainLake, models.TerrainRiver, models.TerrainMountain,
		models.TerrainWasteland,
	),
	fireAndIceRow(6,
		models.TerrainMountain, models.TerrainRiver, models.TerrainDesert, models.TerrainMountain,
		models.TerrainLake, models.TerrainWasteland, models.TerrainForest, models.TerrainRiver,
		models.TerrainWasteland, models.TerrainMountain, models.TerrainRiver, models.TerrainSwamp,
	),
	fireAndIceRow(7,
		models.TerrainSwamp, models.TerrainLake, models.TerrainRiver, models.TerrainSwamp,
		models.TerrainPlains, models.TerrainMountain, models.TerrainLake, models.TerrainRiver,
		models.TerrainDesert, models.TerrainSwamp, models.TerrainRiver, models.TerrainWasteland,
		models.TerrainLake,
	),
	fireAndIceRow(8,
		models.TerrainMountain, models.TerrainForest, models.TerrainRiver, models.TerrainWasteland,
		models.TerrainDesert, models.TerrainSwamp, models.TerrainDesert, models.TerrainRiver,
		models.TerrainLake, models.TerrainPlains, models.TerrainRiver, models.TerrainPlains,
	),
}

var mapDefinitions = map[MapID]mapDefinition{
	MapBase: {
		Info: MapInfo{ID: MapBase, Name: "Base"},
		Rows: cloneRows(baseRows),
	},
	MapRevisedBase: {
		Info: MapInfo{ID: MapRevisedBase, Name: "Revised Base"},
		Rows: cloneRows(revisedBaseRows),
	},
	MapLakes: {
		Info: MapInfo{ID: MapLakes, Name: "Lakes"},
		Rows: cloneRows(lakesRows),
	},
	MapFireAndIce: {
		Info: MapInfo{ID: MapFireAndIce, Name: "Fire & Ice"},
		Rows: cloneRows(fireAndIceRows),
	},
	MapArchipelago: {
		Info: MapInfo{ID: MapArchipelago, Name: "Archipelago"},
		Rows: []rowDefinition{
			baseRow(0,
				models.TerrainSwamp, models.TerrainLake, models.TerrainWasteland, models.TerrainForest,
				models.TerrainLake, models.TerrainPlains, models.TerrainRiver, models.TerrainWasteland,
				models.TerrainPlains, models.TerrainSwamp, models.TerrainRiver, models.TerrainLake,
				models.TerrainForest,
			),
			baseRow(1,
				models.TerrainForest, models.TerrainMountain, models.TerrainSwamp, models.TerrainWasteland,
				models.TerrainMountain, models.TerrainDesert, models.TerrainRiver, models.TerrainRiver,
				models.TerrainForest, models.TerrainRiver, models.TerrainSwamp, models.TerrainMountain,
			),
			baseRow(2,
				models.TerrainDesert, models.TerrainSwamp, models.TerrainDesert, models.TerrainPlains,
				models.TerrainLake, models.TerrainForest, models.TerrainRiver, models.TerrainLake,
				models.TerrainRiver, models.TerrainRiver, models.TerrainRiver, models.TerrainRiver,
				models.TerrainLake,
			),
			baseRow(3,
				models.TerrainRiver, models.TerrainMountain, models.TerrainWasteland, models.TerrainRiver,
				models.TerrainRiver, models.TerrainRiver, models.TerrainPlains, models.TerrainDesert,
				models.TerrainRiver, models.TerrainWasteland, models.TerrainDesert, models.TerrainRiver,
			),
			baseRow(4,
				models.TerrainLake, models.TerrainRiver, models.TerrainForest, models.TerrainRiver,
				models.TerrainWasteland, models.TerrainRiver, models.TerrainRiver, models.TerrainWasteland,
				models.TerrainForest, models.TerrainRiver, models.TerrainPlains, models.TerrainSwamp,
				models.TerrainPlains,
			),
			baseRow(5,
				models.TerrainWasteland, models.TerrainRiver, models.TerrainRiver, models.TerrainMountain,
				models.TerrainSwamp, models.TerrainRiver, models.TerrainRiver, models.TerrainRiver,
				models.TerrainRiver, models.TerrainMountain, models.TerrainForest, models.TerrainRiver,
			),
			baseRow(6,
				models.TerrainMountain, models.TerrainPlains, models.TerrainForest, models.TerrainLake,
				models.TerrainDesert, models.TerrainLake, models.TerrainRiver, models.TerrainMountain,
				models.TerrainSwamp, models.TerrainRiver, models.TerrainRiver, models.TerrainRiver,
				models.TerrainLake,
			),
			baseRow(7,
				models.TerrainSwamp, models.TerrainMountain, models.TerrainDesert, models.TerrainWasteland,
				models.TerrainMountain, models.TerrainForest, models.TerrainRiver, models.TerrainPlains,
				models.TerrainRiver, models.TerrainDesert, models.TerrainSwamp, models.TerrainWasteland,
			),
			baseRow(8,
				models.TerrainLake, models.TerrainDesert, models.TerrainForest, models.TerrainPlains,
				models.TerrainSwamp, models.TerrainPlains, models.TerrainRiver, models.TerrainDesert,
				models.TerrainMountain, models.TerrainRiver, models.TerrainWasteland, models.TerrainPlains,
				models.TerrainDesert,
			),
		},
	},
	MapFjords: {
		Info: MapInfo{ID: MapFjords, Name: "Fjords"},
		Rows: []rowDefinition{
			baseRow(0,
				models.TerrainForest, models.TerrainSwamp, models.TerrainRiver, models.TerrainPlains,
				models.TerrainDesert, models.TerrainMountain, models.TerrainSwamp, models.TerrainMountain,
				models.TerrainDesert, models.TerrainWasteland, models.TerrainSwamp, models.TerrainLake,
				models.TerrainDesert,
			),
			baseRow(1,
				models.TerrainLake, models.TerrainPlains, models.TerrainRiver, models.TerrainLake,
				models.TerrainForest, models.TerrainWasteland, models.TerrainRiver, models.TerrainRiver,
				models.TerrainRiver, models.TerrainRiver, models.TerrainRiver, models.TerrainPlains,
			),
			baseRow(2,
				models.TerrainMountain, models.TerrainForest, models.TerrainWasteland, models.TerrainRiver,
				models.TerrainRiver, models.TerrainPlains, models.TerrainRiver, models.TerrainSwamp,
				models.TerrainMountain, models.TerrainPlains, models.TerrainDesert, models.TerrainRiver,
				models.TerrainMountain,
			),
			baseRow(3,
				models.TerrainRiver, models.TerrainRiver, models.TerrainRiver, models.TerrainMountain,
				models.TerrainRiver, models.TerrainRiver, models.TerrainForest, models.TerrainWasteland,
				models.TerrainLake, models.TerrainForest, models.TerrainWasteland, models.TerrainRiver,
			),
			baseRow(4,
				models.TerrainWasteland, models.TerrainMountain, models.TerrainDesert, models.TerrainRiver,
				models.TerrainLake, models.TerrainWasteland, models.TerrainRiver, models.TerrainPlains,
				models.TerrainDesert, models.TerrainMountain, models.TerrainPlains, models.TerrainRiver,
				models.TerrainSwamp,
			),
			baseRow(5,
				models.TerrainSwamp, models.TerrainPlains, models.TerrainRiver, models.TerrainForest,
				models.TerrainDesert, models.TerrainForest, models.TerrainRiver, models.TerrainMountain,
				models.TerrainLake, models.TerrainForest, models.TerrainRiver, models.TerrainMountain,
			),
			baseRow(6,
				models.TerrainDesert, models.TerrainLake, models.TerrainRiver, models.TerrainSwamp,
				models.TerrainMountain, models.TerrainSwamp, models.TerrainLake, models.TerrainRiver,
				models.TerrainPlains, models.TerrainSwamp, models.TerrainRiver, models.TerrainForest,
				models.TerrainWasteland,
			),
			baseRow(7,
				models.TerrainForest, models.TerrainRiver, models.TerrainPlains, models.TerrainWasteland,
				models.TerrainPlains, models.TerrainDesert, models.TerrainWasteland, models.TerrainRiver,
				models.TerrainRiver, models.TerrainRiver, models.TerrainWasteland, models.TerrainLake,
			),
			baseRow(8,
				models.TerrainSwamp, models.TerrainRiver, models.TerrainRiver, models.TerrainForest,
				models.TerrainLake, models.TerrainMountain, models.TerrainLake, models.TerrainRiver,
				models.TerrainForest, models.TerrainDesert, models.TerrainSwamp, models.TerrainPlains,
				models.TerrainDesert,
			),
		},
	},
}

var mapCoordinateIndexes = buildMapCoordinateIndexes()

func buildMapCoordinateIndexes() map[MapID]coordinateIndex {
	indexes := make(map[MapID]coordinateIndex, len(mapDefinitions))
	for id, def := range mapDefinitions {
		indexes[id] = buildCoordinateIndex(def)
	}
	return indexes
}

func buildCoordinateIndex(def mapDefinition) coordinateIndex {
	index := coordinateIndex{
		displayByHex:   make(map[Hex]string),
		hexByDisplayID: make(map[string]Hex),
	}
	for _, row := range def.Rows {
		landIndex := 0
		rowLabel := rowLabelForIndex(row.R)
		for offset, terrain := range row.Terrains {
			if terrain == models.TerrainRiver {
				continue
			}
			landIndex++
			hex := NewHex(row.StartQ+offset, row.R)
			display := fmt.Sprintf("%s%d", rowLabel, landIndex)
			index.displayByHex[hex] = display
			index.hexByDisplayID[normalizeDisplayCoordinate(display)] = hex
		}
	}
	return index
}

func rowLabelForIndex(index int) string {
	if index < 0 {
		return ""
	}

	label := ""
	for {
		label = string(rune('A'+(index%26))) + label
		index = index/26 - 1
		if index < 0 {
			return label
		}
	}
}

func normalizeDisplayCoordinate(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

func CloneCustomMapDefinition(in *CustomMapDefinition) *CustomMapDefinition {
	if in == nil {
		return nil
	}

	out := &CustomMapDefinition{
		Name:            strings.TrimSpace(in.Name),
		RowCount:        in.RowCount,
		FirstRowColumns: in.FirstRowColumns,
		FirstRowLonger:  in.FirstRowLonger,
		Rows:            make([][]models.TerrainType, len(in.Rows)),
	}
	for i, row := range in.Rows {
		out.Rows[i] = append([]models.TerrainType(nil), row...)
	}
	return out
}

func (d CustomMapDefinition) expectedRowLength(row int) int {
	if row%2 == 0 {
		return d.FirstRowColumns
	}
	if d.FirstRowLonger {
		return d.FirstRowColumns - 1
	}
	return d.FirstRowColumns + 1
}

func (d CustomMapDefinition) validate() error {
	if d.RowCount <= 0 {
		return fmt.Errorf("rowCount must be greater than 0")
	}
	if d.FirstRowColumns <= 0 {
		return fmt.Errorf("firstRowColumns must be greater than 0")
	}
	if len(d.Rows) != d.RowCount {
		return fmt.Errorf("rows length %d does not match rowCount %d", len(d.Rows), d.RowCount)
	}
	if d.RowCount > 1 && d.FirstRowLonger && d.FirstRowColumns < 2 {
		return fmt.Errorf("firstRowColumns must be at least 2 when the first row is longer")
	}

	for rowIndex, row := range d.Rows {
		expected := d.expectedRowLength(rowIndex)
		if expected <= 0 {
			return fmt.Errorf("row %d has invalid expected length %d", rowIndex, expected)
		}
		if len(row) != expected {
			return fmt.Errorf("row %d has %d columns, expected %d", rowIndex, len(row), expected)
		}
		for colIndex, terrain := range row {
			if terrain < models.TerrainPlains || terrain > models.TerrainVolcano {
				return fmt.Errorf("row %d column %d has invalid terrain %d", rowIndex, colIndex, terrain)
			}
		}
	}

	return nil
}

func (d CustomMapDefinition) MapInfo() MapInfo {
	name := strings.TrimSpace(d.Name)
	if name == "" {
		name = customMapInfo.Name
	}
	return MapInfo{ID: MapCustom, Name: name}
}

func (d CustomMapDefinition) toMapDefinition() (mapDefinition, error) {
	if err := d.validate(); err != nil {
		return mapDefinition{}, err
	}

	rows := make([]rowDefinition, d.RowCount)
	for rowIndex, terrains := range d.Rows {
		rows[rowIndex] = rowDefinition{
			R:        rowIndex,
			StartQ:   startQForRow(d.FirstRowLonger, rowIndex),
			Terrains: append([]models.TerrainType(nil), terrains...),
		}
	}

	return mapDefinition{
		Info: d.MapInfo(),
		Rows: rows,
	}, nil
}

func definitionForBuiltInMap(id MapID) (mapDefinition, error) {
	def, ok := mapDefinitions[id]
	if !ok {
		return mapDefinition{}, fmt.Errorf("unknown map id: %s", id)
	}
	return def, nil
}

func AvailableMaps() []MapInfo {
	infos := make([]MapInfo, 0, len(mapDefinitions)+1)
	for _, def := range mapDefinitions {
		infos = append(infos, def.Info)
	}
	infos = append(infos, customMapInfo)
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].ID < infos[j].ID
	})
	return infos
}

func NormalizeMapID(raw string) MapID {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return MapBase
	}
	switch normalized {
	case "base", "base game", "base_game":
		return MapBase
	case "lakes", "lake":
		return MapLakes
	case "revised base", "revised-base", "revised_base", "revised base game", "revised-base-game", "revised_base_game":
		return MapRevisedBase
	case "fire_and_ice", "fire and ice", "fire&ice", "fire & ice":
		return MapFireAndIce
	default:
		return MapID(normalized)
	}
}

func MapInfoByID(id MapID) (MapInfo, bool) {
	if id == MapCustom {
		return customMapInfo, true
	}
	def, ok := mapDefinitions[id]
	if !ok {
		return MapInfo{}, false
	}
	return def.Info, true
}

func LayoutForMap(id MapID) (map[Hex]models.TerrainType, error) {
	if id == MapCustom {
		return nil, fmt.Errorf("custom maps require an explicit definition")
	}
	def, ok := mapDefinitions[id]
	if !ok {
		return nil, fmt.Errorf("unknown map id: %s", id)
	}
	return def.Layout(), nil
}

func DisplayCoordinateForHex(id MapID, hex Hex) (string, bool) {
	index, ok := mapCoordinateIndexes[id]
	if !ok {
		return "", false
	}
	display, ok := index.displayByHex[hex]
	return display, ok
}

func HexForDisplayCoordinate(id MapID, display string) (Hex, bool) {
	index, ok := mapCoordinateIndexes[id]
	if !ok {
		return Hex{}, false
	}
	hex, ok := index.hexByDisplayID[normalizeDisplayCoordinate(display)]
	return hex, ok
}
