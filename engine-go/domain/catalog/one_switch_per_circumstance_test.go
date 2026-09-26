package catalog_test

import (
	"encoding/json"
	"testing"

	"t20engine/domain/catalog"
)

// TestNoEntryOffersTwoSwitchesForTheSameCircumstance (ALE-399).
//
// A tela dos situacionais desenha UMA LINHA POR GRUPO, e o agrupamento é por
// FLAG. Um condicional de `context` não tem flag, então cada modificador vira o
// próprio grupo — e dois `context` com a MESMA nota na mesma entrada viram duas
// linhas idênticas, sem nada que as distinga.
//
// Foi o que aconteceu com a Torcida, que dá "+2 em testes de perícia e Defesa"
// (p131) e portanto são dois modificadores. O olho pegou na ficha; nenhum
// limiar pegaria.
//
// O conserto é de DADO: circunstância com mais de um modificador usa `flagOn`
// com flag compartilhada. Este guarda varre o catálogo inteiro para que a
// escolha valha para a entrada que nascer amanhã, e falha com o nome dela — a
// diferença entre "conserte isto" e "procure".
func TestNoEntryOffersTwoSwitchesForTheSameCircumstance(t *testing.T) {
	type modifier struct {
		Condition *struct {
			C    string `json:"c"`
			Note string `json:"note"`
		} `json:"condition"`
	}
	type entry struct {
		ID        string     `json:"id"`
		Name      string     `json:"name"`
		Modifiers []modifier `json:"modifiers"`
	}

	// A DÍVIDA NOMEADA, e ela só pode ENCOLHER. Baluarte e Duelo do cavaleiro
	// têm a mesma forma e desenham a mesma linha dobrada, mas os dois CUSTAM PM
	// e duram até um marco do turno — convertê-los mexeu nos totais com
	// condicionais da ficha `lenda-nv20-maximo` (Defesa 32 → 22), o que quer
	// dizer que a escolha entre `context` e `flagOn` interage com algo que esta
	// fatia não mediu. Ficam registrados na ALE-400 em vez de arrastados junto.
	//
	// Entrada nova com a mesma forma REPROVA: a linha de base não os isenta de
	// existir, só de bloquear a convenção enquanto a investigação não acontece.
	divida := map[string]bool{
		"Baluarte +2": true, "Baluarte +4": true, "Baluarte +6": true,
		"Baluarte +8": true, "Baluarte +10": true,
		"Duelo +2": true, "Duelo +3": true, "Duelo +4": true, "Duelo +5": true,
	}
	medidos, reprovados, naDivida := 0, 0, 0
	for _, recurso := range []string{
		"granted-powers", "general-powers", "class-powers", "race-defs", "items", "origins",
	} {
		raw, ok := catalog.Resource(recurso)
		if !ok {
			t.Fatalf("recurso %q não está registrado — o caso mediria menos do que diz", recurso)
		}
		var entries []entry
		if json.Unmarshal(raw, &entries) != nil {
			continue // arquivo que não é lista de verbetes com modificador
		}
		for _, e := range entries {
			porNota := map[string]int{}
			for _, m := range e.Modifiers {
				if m.Condition == nil || m.Condition.C != "context" {
					continue
				}
				medidos++
				porNota[m.Condition.Note]++
			}
			for nota, n := range porNota {
				if n < 2 {
					continue
				}
				if divida[e.Name] {
					naDivida++
					continue
				}
				reprovados++
				t.Errorf("%s/%s: %d modificadores de `context` com a nota %q — a tela "+
					"desenha uma linha por grupo, e `context` não agrupa, então saem "+
					"%d interruptores idênticos. Use `flagOn` com uma flag "+
					"compartilhada.", recurso, comoSeChama(e.Name, e.ID), n, nota, n)
			}
		}
	}

	// O CONTROLE: havia `context` para medir. Sem ele, um catálogo que perdesse
	// a chave `condition` passaria verde sobre nada.
	// O CONTROLE: havia `context` para medir. Sem ele, um catálogo que perdesse
	// a chave `condition` passaria verde sobre nada.
	//
	// O piso é 10 e o catálogo tem 31. Ele nasceu em 20 e reprovou o próprio
	// conserto: converter Baluarte e Duelo para `flagOn` derrubou a contagem
	// para 13, e o controle acusou "só 13". Um piso calibrado no mundo
	// pré-conserto reprova quem conserta — por isso ele fica FOLGADO, longe
	// tanto do zero quanto do número de hoje.
	if medidos < 10 {
		t.Fatalf("só %d modificadores de `context` no catálogo — eram 31 quando isto "+
			"foi escrito, e o caso não estaria medindo nada", medidos)
	}
	// A dívida também tem denominador: se ela encolher e esta lista não, o caso
	// avisa — é a mesma catraca do teto de linha.
	if naDivida != len(divida) {
		t.Errorf("a linha de base tem %d entradas e só %d ainda dobram a linha — "+
			"tire da lista as que foram consertadas", len(divida), naDivida)
	}
	if reprovados == 0 {
		t.Logf("interruptores de contexto conferidos: %d | na dívida: %d", medidos, naDivida)
	}
}

// comoSeChama: nem todo verbete do catálogo tem `name` — o de origem, por
// exemplo, é indexado por id. A mensagem precisa de um dos dois para nomear o
// caso em vez de mandar procurar.
func comoSeChama(nome, id string) string {
	if nome != "" {
		return nome
	}
	return id
}
