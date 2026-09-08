package main

import (
	"encoding/json"
	"sort"
	"testing"
)

// TODO CAMPO DO `create` É CLASSIFICADO: ou ele aponta para o catálogo e é
// conferido, ou ele não aponta e está escrito aqui (ALE-226).
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
// # Ele já pagou por si
//
// Escrevendo a lista abaixo apareceram DOIS campos que o validador não conferia
// e eu não tinha visto: o `size` — o catálogo tem a lista de tamanhos — e as
// CHAVES do `classChoices`, que são nomes de classe (`{"Arcanista": {…}}`).
// Referência escondida em chave de objeto não se parece com referência, e foi
// justamente a que escapou.
func TestEveryCreateFieldOfTheSeedIsClassified(t *testing.T) {
	// APONTAM para o catálogo, e o `validateCatalogRefs` confere cada um.
	// Mexeu aqui? Mexa lá — e o `TestTheSeedRefusesEveryUnknownCatalogReference`
	// é quem prova que a conferência morde.
	doCatalogo := map[string]bool{
		"races": true, "origin": true, "classes": true, "god": true,
		"godPower": true, "items": true, "size": true, "classChoices": true,
	}
	// NÃO apontam: são números, texto livre do dono, ou nomes de ATRIBUTO, que
	// são do motor e não do catálogo.
	proprios := map[string]bool{
		"name": true, "displacement": true, "raceAttributeChoices": true,
		"strength": true, "dexterity": true, "constitution": true,
		"intelligence": true, "wisdom": true, "charisma": true,
	}

	var sf seedFile
	if err := json.Unmarshal(seedData, &sf); err != nil {
		t.Fatalf("seed-data.json: %v", err)
	}
	vistos, campos := map[string]bool{}, 0
	for _, u := range sf.Users {
		for _, ch := range u.Characters {
			var criar map[string]json.RawMessage
			if err := json.Unmarshal(ch.Create, &criar); err != nil {
				t.Fatalf("create: %v", err)
			}
			for campo := range criar {
				campos++
				vistos[campo] = true
			}
		}
	}
	naoClassificados := []string{}
	for campo := range vistos {
		if !doCatalogo[campo] && !proprios[campo] {
			naoClassificados = append(naoClassificados, campo)
		}
	}
	sort.Strings(naoClassificados)
	for _, campo := range naoClassificados {
		t.Errorf("o `create` do seed tem o campo %q e este guarda não sabe o que ele é.\n"+
			"    Se ele aponta para o catálogo, acrescente a conferência no `validateCatalogRefs` e o nome em `doCatalogo`.\n"+
			"    Se não aponta, o nome vai em `proprios`. Campo novo nasce DESCOBERTO, e é assim que o seed volta a mentir.",
			campo)
	}

	// O DENOMINADOR. Sem ele, um `seed-data.json` que deixasse de ter `create`
	// nenhum passaria verde — e "nenhum campo não classificado" e "nenhum campo"
	// são a mesma lista vazia.
	if campos < 100 {
		t.Fatalf("o guarda viu só %d campos de `create` em todo o seed: ele parou de achar o que veio medir", campos)
	}
	// E o outro lado: os campos que EU classifiquei como do catálogo têm de
	// existir mesmo no arquivo. Um nome sobrando em `doCatalogo` é conferência
	// escrita para um campo que ninguém usa, e ela apodrece calada.
	for campo := range doCatalogo {
		if !vistos[campo] {
			t.Errorf("`doCatalogo` lista %q, que não existe em nenhum `create` do seed: "+
				"ou o campo foi removido do arquivo, ou o nome está errado aqui", campo)
		}
	}
}
