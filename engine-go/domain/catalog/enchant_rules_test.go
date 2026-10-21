package catalog

import (
	"encoding/json"
	"sort"
	"testing"
)

// OS ENCANTOS, pelo que sobrevive SEM o livro na mão (ALE-411, ALE-439).
//
// São DUAS TABELAS e os mesmos três guardas: a 8-8 é de armas (p335-336) e a
// 8-10 é de armaduras e escudos (p338-339). Elas entram como casos de um
// `t.Run` em vez de dois arquivos — a regra é a mesma e uma cópia divergiria na
// primeira correção.
//
// A conferência contra a página é do `scripts/audit-enchants.py`, que lê o PDF
// — e o PDF vive fora do repositório, então ele não roda na CI. O que fica aqui
// é o que o catálogo prova sozinho, e é mais do que parece: a Tabela 8-8 é uma
// tabela de ROLAGEM, e uma tabela de rolagem tem denominador embutido.

// tabelaDeEncanto é uma das duas tabelas de rolagem do livro.
type tabelaDeEncanto struct {
	categoria string
	tabela    string
	pagina    string
	// ultimoNoDado é onde a faixa dos encantos acaba: de lá até 100 as duas
	// tabelas mandam rolar em OUTRA ("arma específica", "item específico"), e o
	// livro diz que nesse caso o item PERDE os encantos rolados.
	ultimoNoDado int
	// preRequisitos é quantos verbetes da tabela escrevem "Pré-requisito:", e
	// `quaisPreRequisitos` diz QUAIS — o número sozinho não ensina nada a quem
	// o vir reprovar.
	preRequisitos      int
	quaisPreRequisitos string
}

var tabelasDeEncanto = []tabelaDeEncanto{
	{
		categoria: "weapon-enchant", tabela: "8-8", pagina: "p336", ultimoNoDado: 90,
		preRequisitos: 3,
		quaisPreRequisitos: "Energética e Magnífica pedem formidável, e Lancinante " +
			"pede dilacerante (p335-336)",
	},
	{
		categoria: "armor-enchant", tabela: "8-10", pagina: "p339", ultimoNoDado: 90,
		preRequisitos:      1,
		quaisPreRequisitos: "o Guardião pede defensor (p339)",
	},
}

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
func TestTheEnchantsTileTheRollRange(t *testing.T) {
	for _, tabela := range tabelasDeEncanto {
		t.Run(tabela.tabela, func(t *testing.T) {
			encantos := itemsOfCategory[weaponEnchant](t, tabela.categoria)
			sort.Slice(encantos, func(a, b int) bool { return encantos[a].RollMin < encantos[b].RollMin })

			esperado := 1
			for _, e := range encantos {
				if e.RollMin != esperado {
					t.Fatalf("a faixa de %s começa em %d e a anterior terminou em %d: "+
						"há um buraco ou uma sobreposição no d%% da Tabela %s (%s)",
						e.Name, e.RollMin, esperado-1, tabela.tabela, tabela.pagina)
				}
				if e.RollMax < e.RollMin {
					t.Fatalf("a faixa de %s vai de %d a %d, de trás para frente",
						e.Name, e.RollMin, e.RollMax)
				}
				esperado = e.RollMax + 1
			}
			if esperado-1 != tabela.ultimoNoDado {
				t.Errorf("as faixas dos %d encantos terminam em %d e deviam terminar em %d — "+
					"de %d a 100 a tabela manda rolar em OUTRA tabela",
					len(encantos), esperado-1, tabela.ultimoNoDado, tabela.ultimoNoDado+1)
			}
		})
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
func TestEveryEnchantEitherAppliesOrSaysWhyNot(t *testing.T) {
	for _, tabela := range tabelasDeEncanto {
		t.Run(tabela.tabela, func(t *testing.T) { eitherAppliesOrSaysWhyNot(t, tabela) })
	}
}

func eitherAppliesOrSaysWhyNot(t *testing.T, tabela tabelaDeEncanto) {
	t.Helper()
	encantos := itemsOfCategory[weaponEnchant](t, tabela.categoria)
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
			t.Errorf("%s conta como %d encantos, e a Tabela %s só tem 1 e 2 "+
				"(o asterisco é \"conta como dois\")", e.Name, e.CountsAs, tabela.tabela)
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
func TestEveryEnchantPrerequisiteExists(t *testing.T) {
	for _, tabela := range tabelasDeEncanto {
		t.Run(tabela.tabela, func(t *testing.T) { prerequisiteExists(t, tabela) })
	}
}

func prerequisiteExists(t *testing.T, tabela tabelaDeEncanto) {
	t.Helper()
	encantos := itemsOfCategory[weaponEnchant](t, tabela.categoria)
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
	//
	// O número é POR TABELA, e está escrito ao lado de quais são: a 8-8 tem
	// três e a 8-10 tem um.
	if medidos != tabela.preRequisitos {
		t.Errorf("%d encantos da Tabela %s declaram pré-requisito e o livro escreve %d (%s)",
			medidos, tabela.tabela, tabela.preRequisitos, tabela.quaisPreRequisitos)
	}
}
