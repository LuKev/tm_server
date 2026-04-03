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
	MapFjords      MapID = "fjords"
	MapLakes       MapID = "lakes"
	MapRevisedBase MapID = "revised-base"
)

type MapInfo struct {
	ID   MapID  `json:"id"`
	Name string `json:"name"`
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

func mapStartQForRow(r int) int {
	if r%2 == 0 {
		return -(r / 2)
	}
	return -((r - 1) / 2)
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
	baseRow(0,
		models.TerrainMountain, models.TerrainLake, models.TerrainWasteland, models.TerrainPlains,
		models.TerrainDesert, models.TerrainLake, models.TerrainDesert, models.TerrainWasteland,
		models.TerrainRiver, models.TerrainRiver, models.TerrainForest, models.TerrainLake,
	),
	baseRow(1,
		models.TerrainDesert, models.TerrainSwamp, models.TerrainForest, models.TerrainRiver,
		models.TerrainSwamp, models.TerrainPlains, models.TerrainRiver, models.TerrainForest,
		models.TerrainMountain, models.TerrainRiver, models.TerrainPlains, models.TerrainRiver,
		models.TerrainSwamp,
	),
	baseRow(2,
		models.TerrainPlains, models.TerrainRiver, models.TerrainForest, models.TerrainWasteland,
		models.TerrainMountain, models.TerrainRiver, models.TerrainSwamp, models.TerrainLake,
		models.TerrainWasteland, models.TerrainRiver, models.TerrainRiver, models.TerrainDesert,
	),
	baseRow(3,
		models.TerrainLake, models.TerrainWasteland, models.TerrainMountain, models.TerrainRiver,
		models.TerrainDesert, models.TerrainPlains, models.TerrainForest, models.TerrainRiver,
		models.TerrainRiver, models.TerrainDesert, models.TerrainRiver, models.TerrainSwamp,
		models.TerrainWasteland,
	),
	baseRow(4,
		models.TerrainForest, models.TerrainDesert, models.TerrainRiver, models.TerrainSwamp,
		models.TerrainLake, models.TerrainRiver, models.TerrainRiver, models.TerrainWasteland,
		models.TerrainRiver, models.TerrainMountain, models.TerrainForest, models.TerrainPlains,
	),
	baseRow(5,
		models.TerrainMountain, models.TerrainRiver, models.TerrainPlains, models.TerrainMountain,
		models.TerrainRiver, models.TerrainDesert, models.TerrainRiver, models.TerrainMountain,
		models.TerrainRiver, models.TerrainPlains, models.TerrainSwamp, models.TerrainLake,
		models.TerrainWasteland,
	),
	baseRow(6,
		models.TerrainWasteland, models.TerrainRiver, models.TerrainRiver, models.TerrainWasteland,
		models.TerrainForest, models.TerrainPlains, models.TerrainSwamp, models.TerrainDesert,
		models.TerrainRiver, models.TerrainRiver, models.TerrainMountain, models.TerrainRiver,
	),
	baseRow(7,
		models.TerrainDesert, models.TerrainLake, models.TerrainSwamp, models.TerrainRiver,
		models.TerrainLake, models.TerrainMountain, models.TerrainLake, models.TerrainRiver,
		models.TerrainRiver, models.TerrainMountain, models.TerrainForest, models.TerrainRiver,
		models.TerrainLake,
	),
	baseRow(8,
		models.TerrainSwamp, models.TerrainPlains, models.TerrainRiver, models.TerrainForest,
		models.TerrainRiver, models.TerrainRiver, models.TerrainRiver, models.TerrainForest,
		models.TerrainWasteland, models.TerrainPlains, models.TerrainDesert, models.TerrainSwamp,
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
				models.TerrainRiver, models.TerrainForest, models.TerrainWasteland, models.TerrainLake,
				models.TerrainForest, models.TerrainWasteland, models.TerrainRiver, models.TerrainRiver,
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
				models.TerrainForest, models.TerrainRiver, models.TerrainRiver, models.TerrainPlains,
				models.TerrainWasteland, models.TerrainPlains, models.TerrainDesert, models.TerrainWasteland,
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

func AvailableMaps() []MapInfo {
	infos := make([]MapInfo, 0, len(mapDefinitions))
	for _, def := range mapDefinitions {
		infos = append(infos, def.Info)
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].ID < infos[j].ID
	})
	return infos
}

func NormalizeMapID(raw string) MapID {
	id := MapID(strings.ToLower(strings.TrimSpace(raw)))
	if id == "" {
		return MapBase
	}
	return id
}

func MapInfoByID(id MapID) (MapInfo, bool) {
	def, ok := mapDefinitions[id]
	if !ok {
		return MapInfo{}, false
	}
	return def.Info, true
}

func LayoutForMap(id MapID) (map[Hex]models.TerrainType, error) {
	def, ok := mapDefinitions[id]
	if !ok {
		return nil, fmt.Errorf("unknown map id: %s", id)
	}
	return def.Layout(), nil
}
