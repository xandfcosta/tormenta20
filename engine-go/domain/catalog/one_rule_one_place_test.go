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

	// A DÍVIDA ACABOU (ALE-403). Ela nomeava nove poderes cuja regra estava
	// escrita de duas a quatorze vezes; hoje cada um é um verbete e as 31
	// concessões apontam. O mapa fica vazio de propósito, e não some: é ele que
	// faz "consertar sem tirar da lista" reprovar, e um dia alguém vai
	// precisar registrar uma dívida nova aqui.
	divida := map[string]bool{}
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

// O VERBETE NÃO PODE CARREGAR MODIFICADOR (ALE-403).
//
// A divisão entre verbete e concessão é resolvida no `domain/book`, porque são
// os leitores DELE que precisam de nome e descrição. O MOTOR não resolve: ele
// lê o despejo cru e, para poder de classe, só olha `Modifiers`.
//
// Isso funciona porque nenhum dos nove poderes divididos tem modificador. É uma
// coincidência feliz, não uma garantia — e sem este caso, o dia em que alguém
// desse um `+2` a um verbete o motor o ignoraria em silêncio, enquanto a tela
// mostraria o bônus. A ficha diria uma coisa e o número seria outro.
//
// Quando isso for preciso, o conserto não é afrouxar aqui: é o motor passar a
// resolver o `powerUid`, como já faz com o benefício de origem.
func TestNoClassPowerEntryCarriesModifiersTheEngineWouldIgnore(t *testing.T) {
	raw, ok := catalog.Resource("class-powers")
	if !ok {
		t.Fatal("catálogo de poderes de classe ausente")
	}
	var powers []map[string]any
	if err := json.Unmarshal(raw, &powers); err != nil {
		t.Fatalf("poderes de classe: %v", err)
	}

	verbetes := 0
	for _, p := range powers {
		if _, éConcessão := p["className"]; éConcessão {
			continue
		}
		verbetes++
		mods, tem := p["modifiers"].([]any)
		if !tem || len(mods) == 0 {
			continue
		}
		nome, _ := p["name"].(string)
		t.Errorf("o verbete %q tem %d modificador(es), e o MOTOR não resolve "+
			"`powerUid` para poder de classe — ele os ignoraria, e a tela mostraria "+
			"um bônus que a ficha não tem", nome, len(mods))
	}

	// O CONTROLE: havia verbete para medir. Zero verbetes passaria verde sobre
	// nada, e é o que aconteceria se a divisão fosse desfeita.
	if verbetes != 9 {
		t.Errorf("achei %d verbetes e a divisão criou 9 — se mudou de propósito, "+
			"mude o número; se não, alguém desfez a divisão", verbetes)
	}
}
