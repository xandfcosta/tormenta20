package catalog

import (
	"encoding/json"
	"sort"
	"testing"
)

// OS ENCANTOS DE ARMA, pelo que sobrevive SEM o livro na mão (ALE-411).
//
// A conferência contra a página é do `scripts/audit-enchants.py`, que lê o PDF
// — e o PDF vive fora do repositório, então ele não roda na CI. O que fica aqui
// é o que o catálogo prova sozinho, e é mais do que parece: a Tabela 8-8 é uma
// tabela de ROLAGEM, e uma tabela de rolagem tem denominador embutido.

type weaponEnchant struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Category    string          `json:"category"`
	RollMin     int             `json:"rollMin"`
	RollMax     int             `json:"rollMax"`
	CountsAs    int             `json:"countsAs"`
	Description string          `json:"description"`
	Requires    string          `json:"requires"`
	Unmodeled   string          `json:"unmodeled"`
	Modifiers   json.RawMessage `json:"modifiers"`
}

// ultimoEncantoNoDado é onde a faixa dos encantos acaba. De 91 a 100 a Tabela
// 8-8 manda rolar na Tabela 8-9 ("Arma específica"), e o próprio livro diz que
// nesse caso o item PERDE os encantos rolados — não é um encanto, e por isso
// não está no catálogo.
const ultimoEncantoNoDado = 90

func weaponEnchants(t *testing.T) []weaponEnchant {
	t.Helper()
	raw, err := files.ReadFile("data/items.json")
	if err != nil {
		t.Fatalf("ler items.json: %v", err)
	}
	var todos []weaponEnchant
	if err := json.Unmarshal(raw, &todos); err != nil {
		t.Fatalf("items.json não é JSON: %v", err)
	}
	encantos := []weaponEnchant{}
	for _, item := range todos {
		if item.Category == "weapon-enchant" {
			encantos = append(encantos, item)
		}
	}
	if len(encantos) == 0 {
		t.Fatal("nenhum encanto de arma no catálogo: não há o que medir, e " +
			"verde aqui não valeria nada")
	}
	return encantos
}

// A FAIXA DE d% LADRILHA 1..90, e é o denominador de graça desta tabela.
//
// Uma lista de encantos não tem como se conferir sozinha — 26 entradas com
// nomes certos e regras certas se parecem com 28 no terminal. A faixa do dado
// tem: ela é uma DECOMPOSIÇÃO do intervalo, e a soma das parcelas tem de fechar.
//
// Foi exatamente isto que pegou o leitor do PDF perdendo duas linhas: a limpeza
// de marca d'água do `t20pdf.py` descarta o que casa com `^\d{1,3}$`, que é o
// padrão de número de página — e comeu as faixas de UM valor, a Energética (46)
// e a Lancinante (64). O relatório saiu com 26 nomes certos e nada reclamou.
func TestTheWeaponEnchantsTileTheRollRange(t *testing.T) {
	encantos := weaponEnchants(t)
	sort.Slice(encantos, func(a, b int) bool { return encantos[a].RollMin < encantos[b].RollMin })

	esperado := 1
	for _, e := range encantos {
		if e.RollMin != esperado {
			t.Fatalf("a faixa de %s começa em %d e a anterior terminou em %d: "+
				"há um buraco ou uma sobreposição no d%% da Tabela 8-8 (p336)",
				e.Name, e.RollMin, esperado-1)
		}
		if e.RollMax < e.RollMin {
			t.Fatalf("a faixa de %s vai de %d a %d, de trás para frente",
				e.Name, e.RollMin, e.RollMax)
		}
		esperado = e.RollMax + 1
	}
	if esperado-1 != ultimoEncantoNoDado {
		t.Errorf("as faixas dos %d encantos terminam em %d e deviam terminar em %d — "+
			"de %d a 100 a tabela manda rolar em OUTRA tabela",
			len(encantos), esperado-1, ultimoEncantoNoDado, ultimoEncantoNoDado+1)
	}
}

// TER MODIFICADOR E DIZER POR QUE NÃO FOI MODELADO SÃO EXCLUSIVOS.
//
// É o denominador do que o motor NÃO aplica, e ele é escrito como um PERMITIDOS
// e não como uma lista de proibidos: todo encanto sem modificador diz o motivo,
// e quem entrar amanhã sem nenhum dos dois reprova pelo nome.
//
// A metade inversa vale igual e é a que apodrece sozinha: um encanto que ganhe
// modificador e mantenha o motivo antigo passa a mentir sobre si mesmo, com cara
// de registro cuidadoso.
func TestEveryWeaponEnchantEitherAppliesOrSaysWhyNot(t *testing.T) {
	encantos := weaponEnchants(t)
	comModificador, semModificador := 0, 0

	for _, e := range encantos {
		aplica := len(e.Modifiers) > 2 // "[]" é o vazio
		switch {
		case aplica && e.Unmodeled != "":
			t.Errorf("%s tem modificador E diz que não foi modelado (%q) — "+
				"uma das duas coisas envelheceu, e a que mente é o texto",
				e.Name, e.Unmodeled)
		case !aplica && e.Unmodeled == "":
			t.Errorf("%s não tem modificador NEM diz por quê.\n"+
				"Encanto que o motor não aplica entra com o MOTIVO escrito: sem ele, "+
				"\"transcrevi os encantos\" e \"transcrevi treze dos vinte e oito\" "+
				"têm a mesma cara.", e.Name)
		case aplica:
			comModificador++
		default:
			semModificador++
		}
		if e.Description == "" {
			t.Errorf("%s entrou sem a regra do livro em prosa", e.Name)
		}
		if e.CountsAs != 1 && e.CountsAs != 2 {
			t.Errorf("%s conta como %d encantos, e a Tabela 8-8 só tem 1 e 2 "+
				"(o asterisco é \"conta como dois\")", e.Name, e.CountsAs)
		}
	}
	if comModificador+semModificador != len(encantos) {
		t.Fatalf("o denominador não fecha: %d encantos, %d com modificador, %d sem",
			len(encantos), comModificador, semModificador)
	}
	t.Logf("encantos medidos: %d — %d o motor aplica, %d dizem por que não",
		len(encantos), comModificador, semModificador)
}

// O PRÉ-REQUISITO APONTA PARA UM ENCANTO QUE EXISTE.
//
// Três verbetes o escrevem ("Pré-requisito: formidável"), e um id que não existe
// é um ponteiro morto que nada mais acusa — o Go não o vê, e a tela mostraria
// vazio.
func TestEveryWeaponEnchantPrerequisiteExists(t *testing.T) {
	encantos := weaponEnchants(t)
	existe := map[string]bool{}
	for _, e := range encantos {
		existe[e.ID] = true
	}
	medidos := 0
	for _, e := range encantos {
		if e.Requires == "" {
			continue
		}
		medidos++
		if !existe[e.Requires] {
			t.Errorf("%s exige %q, e não há encanto com esse id", e.Name, e.Requires)
		}
	}
	// O NÚMERO, e não "mais que zero": com o piso em zero, este guarda passou
	// verde sobre DOIS pré-requisitos quando a página escreve três — a Magnífica
	// entrou sem o dela porque eu escrevi a chave sem acento, e um denominador
	// frouxo não tem como denunciar o que nunca chegou.
	const osQueOLivroEscreve = 3
	if medidos != osQueOLivroEscreve {
		t.Errorf("%d encantos declaram pré-requisito e a p335-336 escreve %d: "+
			"Energética e Magnífica pedem formidável, e Lancinante pede dilacerante",
			medidos, osQueOLivroEscreve)
	}
}
