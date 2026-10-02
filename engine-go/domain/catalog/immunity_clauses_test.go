package catalog

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"testing"

	"t20engine/domain/engine"
)

// AS CLÁUSULAS DE IMUNIDADE DA p228 moram nas DESCRIÇÕES dos tipos de efeito, e
// este guarda é quem impede o motor de divergir delas.
//
// A regra do `domain/engine/immunity.go` é escrita em Go — duas tabelas pequenas
// lidas pelo tipo da criatura e pela mente dela. Ela PODERIA ter sido derivada do
// texto, mas não foi: ler "Construtos e mortos-vivos são imunes a efeitos de
// cansaço" por regex em produção é um parser de prosa no caminho de uma regra, e
// uma frase reescrita na próxima revisão do catálogo mudaria a regra em silêncio.
//
// O que se varre aqui é o INVERSO, e é barato: toda descrição que FALA em
// imunidade tem de estar coberta pelo motor. Uma cláusula nova no catálogo
// reprova com o id dela, em vez de entrar como texto que ninguém lê.
//
// O DENOMINADOR é a tabela inteira: o caso percorre os 18 tipos e afirma quantos
// olhou, porque "nenhuma cláusula descoberta" e "o arquivo não foi lido" se
// parecem no terminal.

type effectTypeEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// falaEmImunidade casa a palavra em qualquer flexão — "imune", "imunes",
// "imunidade". Larga de propósito: um falso positivo faz o caso PEDIR cobertura
// de uma frase que não é cláusula, e isso se resolve lendo; um falso negativo
// deixa uma regra do livro fora do motor, calada.
var falaEmImunidade = regexp.MustCompile(`(?i)imun`)

func TestEveryImmunityClauseOfTheEffectTypesIsInTheEngine(t *testing.T) {
	bruto, err := files.ReadFile("data/effect-types.json")
	if err != nil {
		t.Fatalf("ler os tipos de efeito: %v", err)
	}
	var tipos []effectTypeEntry
	if err := json.Unmarshal(bruto, &tipos); err != nil {
		t.Fatalf("o catálogo de tipos de efeito está ilegível: %v", err)
	}
	if len(tipos) == 0 {
		t.Fatalf("o catálogo de tipos de efeito veio vazio — o guarda mediria zero cláusulas")
	}

	// O MOTOR, visto de fora: a união do que ele concede a um construto sem
	// mente é tudo o que ele sabe imunizar.
	cobertos := map[string]bool{}
	for _, efeito := range engine.ImmunitiesOfCreature("construto", nil) {
		cobertos[efeito] = true
	}

	medidos, comClausula := 0, []string{}
	for _, tipo := range tipos {
		medidos++
		if !falaEmImunidade.MatchString(tipo.Description) {
			continue
		}
		comClausula = append(comClausula, tipo.ID)
		if !cobertos[tipo.ID] {
			t.Errorf("o tipo de efeito %q (%s) diz na p228 que alguém é imune a ele, e o "+
				"motor não imuniza ninguém:\n  %q\n"+
				"Acrescente a cláusula ao `immunityByCreatureType` ou ao "+
				"`immunityOfTheMindless`, conforme ela fale de TIPO ou de MENTE.",
				tipo.ID, tipo.Name, strings.TrimSpace(tipo.Description))
		}
	}
	if medidos != len(tipos) {
		t.Fatalf("o caso mediu %d dos %d tipos de efeito", medidos, len(tipos))
	}

	// E O INVERSO: o motor não pode imunizar contra um tipo que o catálogo não
	// tem. Sem isto, um id escrito errado viraria uma imunidade que nenhuma
	// condição jamais casa — verde sobre nada.
	conhecidos := map[string]bool{}
	for _, tipo := range tipos {
		conhecidos[tipo.ID] = true
	}
	for efeito := range cobertos {
		if !conhecidos[efeito] {
			t.Errorf("o motor imuniza contra %q, que não é um tipo de efeito do catálogo", efeito)
		}
	}

	sort.Strings(comClausula)
	t.Logf("tipos de efeito: %d | com cláusula de imunidade: %d (%s)",
		medidos, len(comClausula), strings.Join(comClausula, ", "))
}
