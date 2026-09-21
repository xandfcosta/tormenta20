package main

import (
	"encoding/json"
	"sort"
	"testing"
)

// TODO CAMPO DO `create` É CLASSIFICADO: ou ele aponta para o catálogo e é
// conferido, ou ele não aponta e está escrito aqui.
//
// # Por que a lista de campos não basta
//
// O validador confere os campos que EU conhecia quando o escrevi. O modo de
// falhar desta família não é o campo errado — é o campo NOVO: alguém acrescenta
// um campo ao `seed-data.json` que aponta para o catálogo, ninguém lembra do
// validador, e a referência nasce descoberta. Em silêncio, que é a marca do
// defeito que esta issue fecha.
//
// Este guarda inverte o ônus: um campo novo REPROVA até que alguém diga de que
// lado ele está. Classificar é uma linha; descobrir que o seed mentiu é uma
// tarde.
//
// A referência que mais escapa é a escondida em CHAVE de objeto — as do
// `classChoices` são nomes de classe (`{"Arcanista": {…}}`) —, porque ela não se
// parece com referência.
func TestEveryCreateFieldOfTheSeedIsClassified(t *testing.T) {
	// APONTAM para o catálogo, e o `validateCatalogRefs` confere cada um.
	// Mexeu aqui? Mexa lá — e o `TestTheSeedRefusesEveryUnknownCatalogReference`
	// é quem prova que a conferência morde.
	fromCatalog := map[string]bool{
		"races": true, "origin": true, "classes": true, "god": true,
		"godPower": true, "items": true, "size": true, "classChoices": true,
	}
	// NÃO apontam: são números, texto livre do dono, ou nomes de ATRIBUTO, que
	// são do motor e não do catálogo.
	own := map[string]bool{
		"name": true, "displacement": true, "raceAttributeChoices": true,
		"strength": true, "dexterity": true, "constitution": true,
		"intelligence": true, "wisdom": true, "charisma": true,
	}

	var sf seedFile
	if err := json.Unmarshal(seedData, &sf); err != nil {
		t.Fatalf("seed-data.json: %v", err)
	}
	seen, fields := map[string]bool{}, 0
	for _, u := range sf.Users {
		for _, ch := range u.Characters {
			var create map[string]json.RawMessage
			if err := json.Unmarshal(ch.Create, &create); err != nil {
				t.Fatalf("create: %v", err)
			}
			for field := range create {
				fields++
				seen[field] = true
			}
		}
	}
	unclassified := []string{}
	for field := range seen {
		if !fromCatalog[field] && !own[field] {
			unclassified = append(unclassified, field)
		}
	}
	sort.Strings(unclassified)
	for _, field := range unclassified {
		t.Errorf("o `create` do seed tem o campo %q e este guarda não sabe o que ele é.\n"+
			"    Se ele aponta para o catálogo, acrescente a conferência no `validateCatalogRefs` e o nome em `doCatalogo`.\n"+
			"    Se não aponta, o nome vai em `proprios`. Campo novo nasce DESCOBERTO, e é assim que o seed volta a mentir.",
			field)
	}

	// O DENOMINADOR. Sem ele, um `seed-data.json` que deixasse de ter `create`
	// nenhum passaria verde — e "nenhum campo não classificado" e "nenhum campo"
	// são a mesma lista vazia.
	if fields < 100 {
		t.Fatalf("o guarda viu só %d campos de `create` em todo o seed: ele parou de achar o que veio medir", fields)
	}
	// E o outro lado: os campos que EU classifiquei como do catálogo têm de
	// existir mesmo no arquivo. Um nome sobrando em `fromCatalog` é conferência
	// escrita para um campo que ninguém usa, e ela apodrece calada.
	for field := range fromCatalog {
		if !seen[field] {
			t.Errorf("`doCatalogo` lista %q, que não existe em nenhum `create` do seed: "+
				"ou o campo foi removido do arquivo, ou o nome está errado aqui", field)
		}
	}
}
