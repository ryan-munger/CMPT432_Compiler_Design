package internal

import (
	"fmt"
	"sort"
	"strings"
)

type SymbolEntry struct {
	name     string
	dataType string
	position Location
	isInit   bool
	beenUsed bool
}

type SymbolTable struct {
	scopeID     string
	entries     map[string]*SymbolEntry // id to table entry
	subTables   []*SymbolTable
	parentTable *SymbolTable
}

type SymbolTableTree struct {
	rootTable *SymbolTable
}

func NewSymbolTable(scopeID string, parent *SymbolTable) *SymbolTable {
	// must init map or can't add to it
	var newTable *SymbolTable = &SymbolTable{scopeID: scopeID, entries: make(map[string]*SymbolEntry), parentTable: parent}
	return newTable
}

func NewTableEntry(name string, dataType string, pos Location) *SymbolEntry {
	return &SymbolEntry{name: name, dataType: dataType, position: pos, isInit: false, beenUsed: false}
}

func (table *SymbolTable) AddSubTable(subTable *SymbolTable) {
	subTable.parentTable = table
	table.subTables = append(table.subTables, subTable)
}

func (table *SymbolTable) AddEntry(id string, entry *SymbolEntry) {
	table.entries[id] = entry
}

func (table *SymbolTable) EntryExists(id string) bool {
	_, exists := table.entries[id]
	return exists
}

func (stt *SymbolTableTree) ToString() string {
	var sb strings.Builder

	// Table headers
	sb.WriteString(fmt.Sprintf("| %-5s | %-4s | %-7s | %-9s | %-5s | %-5s |\n",
		"Scope", "Name", "Type", "Position", "Init?", "Used?"))
	sb.WriteString(strings.Repeat("-", 54) + "\n")

	// Gather entries
	stt.rootTable.collectEntries(&sb)
	return sb.String()
}

func (table *SymbolTable) collectEntries(sb *strings.Builder) {
	if table == nil {
		return
	}

	// Convert map to slice for sorting - for my eyeballs to see alphabetized
	entrySlice := make([]*SymbolEntry, 0, len(table.entries))
	for _, entry := range table.entries {
		entrySlice = append(entrySlice, entry)
	}

	// Sort slice by name ASC
	sort.Slice(entrySlice, func(i, j int) bool {
		return entrySlice[i].name < entrySlice[j].name
	})

	for _, entry := range entrySlice {
		var pos string = fmt.Sprintf("(%d:%d)", entry.position.line, entry.position.startPos)
		sb.WriteString(fmt.Sprintf("| %-5s | %-4s | %-7s | %-9s | %-5t | %-5t |\n",
			table.scopeID, entry.name, entry.dataType, pos, entry.isInit, entry.beenUsed))
		sb.WriteString(strings.Repeat("-", 54) + "\n")
	}

	for _, subTable := range table.subTables {
		subTable.collectEntries(sb)
	}
}
