package internal

import (
	"fmt"
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

func NewSymbolTable(scopeID string) *SymbolTable {
	return &SymbolTable{scopeID: scopeID}
}

func (table *SymbolTable) AddSubTable(subTable *SymbolTable) {
	subTable.parentTable = table
	table.subTables = append(table.subTables, subTable)
}

func (stt *SymbolTableTree) ToString() string {
	var sb strings.Builder
	// headers
	sb.WriteString("Scope\tName\tType\tLine\tStartPos\tInitialized\tUsed\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	// gather entries
	stt.rootTable.collectEntries(&sb)
	return sb.String()
}

func (table *SymbolTable) collectEntries(sb *strings.Builder) {
	if table == nil {
		return
	}
	for _, entry := range table.entries {
		sb.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%t\t%t\n",
			table.scopeID, entry.name, entry.dataType, entry.position.line, entry.position.startPos, entry.isInit, entry.beenUsed))
	}
	for _, subTable := range table.subTables {
		subTable.collectEntries(sb)
	}
}
