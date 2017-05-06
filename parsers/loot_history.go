package parsers

import (
	"regexp"
	"sort"
)

type LootHistory struct {
	items []LootItem
	lines []int
}

func (r *LootHistory) Name() string {
	return "loot_history"
}

func (r *LootHistory) Lines() []int {
	return r.lines
}

type LootItem struct {
	time       string
	playerName string
	quantity   int64
	name       string
}

var reLootHistory = regexp.MustCompile(`(\d\d:\d\d:\d\d) ([\S ]+) has looted ([\d,'\.]+) x ([\S ]+)$`)

func ParseLootHistory(input Input) (ParserResult, Input) {
	lootHistory := &LootHistory{}
	matches, rest := regexParseLines(reLootHistory, input)
	lootHistory.lines = regexMatchedLines(matches)
	for _, match := range matches {
		lootHistory.items = append(lootHistory.items,
			LootItem{
				time:       match[1],
				playerName: match[2],
				quantity:   ToInt(match[3]),
				name:       match[4],
			})
	}
	sort.Slice(lootHistory.items, func(i, j int) bool { return lootHistory.items[i].name < lootHistory.items[j].name })
	return lootHistory, rest
}
