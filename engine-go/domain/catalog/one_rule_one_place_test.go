package catalog_test

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"t20engine/domain/catalog"
)

// UMA REGRA MORA NUM LUGAR SÓ (ALE-401).
//
// O `class-powers.json` guarda, na mesma linha, duas coisas de naturezas
// diferentes: o PODER (nome, descrição, modificadores) e a CONCESSÃO (qual
// classe o dá, em que nível). Quando dois ou mais concedem o mesmo poder, a
// regra é copiada — "Aumento de Atributo" tem QUATORZE cópias byte a byte
// iguais, uma por classe.
//
// Elas ainda não divergiram. Nada as impede: são quatorze lugares onde alguém
// pode corrigir um e esquecer treze, que é exatamente o caminho que
// `general-powers` × `origins` percorreu até ter dezenove poderes com duas
// regras.
//
// # O que NÃO é duplicação
//
// Dezoito nomes se repetem legitimamente, porque o LIVRO batizou coisas
// diferentes com o mesmo nome: "Magias (1° círculo)" do Arcanista lança
// arcanas com o atributo-chave do caminho, e a do Clérigo lança divinas
// somando Sabedoria. Este caso compara o CORPO da regra, não o nome — só
// reprova quando as cópias dizem a mesma coisa.
func TestNoClassPowerRuleIsWrittenTwice(t *testing.T) {
	raw, ok := catalog.Resource("class-powers")
	if !ok {
		t.Fatal("catálogo de poderes de classe ausente")
	}
	var powers []map[string]any
	if err := json.Unmarshal(raw, &powers); err != nil {
		t.Fatalf("poderes de classe: %v", err)
	}
	if len(powers) < 400 {
		t.Fatalf("só %d poderes de classe — eram 462 quando isto foi escrito, e o "+
			"caso não estaria medindo nada", len(powers))
	}

	// A CONCESSÃO é o que pode variar entre cópias do mesmo poder; o resto é a
	// regra, e ela não pode.
	daConcessao := map[string]bool{
		"id": true, "uid": true, "className": true,
		"grantedAtLevel": true, "grantedByChoice": true, "powerUid": true,
	}
	porRegra := map[string][]string{}
	for _, p := range powers {
		nome, _ := p["name"].(string)
		if p["powerUid"] != nil {
			continue // já APONTA: a regra não está aqui
		}
		regra := map[string]any{}
		for k, v := range p {
			if !daConcessao[k] {
				regra[k] = v
			}
		}
		corpo, err := json.Marshal(regra)
		if err != nil {
			t.Fatalf("%s: %v", nome, err)
		}
		classe, _ := p["className"].(string)
		porRegra[string(corpo)] = append(porRegra[string(corpo)], classe)
	}

	// A DÍVIDA NOMEADA, e ela só pode ENCOLHER (ALE-403).
	//
	// Apontar de verdade exige ensinar TRÊS consumidores a resolver o ponteiro —
	// o `engine.Catalogs`, o `book.ClassPowers` e o `class_power_details` —, e
	// isso é fatia própria. Enquanto ela não acontece, estes dez ficam
	// registrados: nome NOVO com regra repetida reprova, e consertar um sem
	// tirá-lo daqui também.
	divida := map[string]bool{
		"Aumento de Atributo": true, "Ímpeto": true, "Esquiva Sobrenatural": true,
		"Evasão": true, "Autoridade Feudal": true, "Valentão": true,
		"Magias (2° círculo)": true, "Magias (3° círculo)": true,
		"Magias (4° círculo)": true,
	}
	// "Magias (5° círculo)" NÃO entra: o Arcanista lança arcanas e o Clérigo
	// divinas, então as duas linhas dizem coisas diferentes. Ela estava na
	// primeira versão desta lista e a catraca do denominador a expulsou —
	// listei dez e só nove repetem.
	repetidas, naDivida := 0, 0
	for corpo, classes := range porRegra {
		if len(classes) < 2 {
			continue
		}
		sort.Strings(classes)
		var regra map[string]any
		_ = json.Unmarshal([]byte(corpo), &regra)
		nome, _ := regra["name"].(string)
		if divida[nome] {
			naDivida++
			continue
		}
		repetidas++
		t.Errorf("%q tem a MESMA regra escrita %d vezes, uma por classe (%s). O poder "+
			"é um só: a linha da classe tem de APONTAR para ele com `powerUid`, e a "+
			"concessão (classe e nível) é que fica aqui",
			nome, len(classes), strings.Join(classes, ", "))
	}
	// A dívida tem denominador: encolher a lista sem consertar, ou consertar sem
	// encolher, reprova — é a catraca do teto de linha.
	if naDivida != len(divida) {
		t.Errorf("a linha de base tem %d nomes e só %d ainda repetem a regra — tire "+
			"da lista os que foram consertados", len(divida), naDivida)
	}
	if repetidas == 0 {
		t.Logf("regras de poder de classe conferidas: %d | na dívida: %d",
			len(porRegra), naDivida)
	}
}
