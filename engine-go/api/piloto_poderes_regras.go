package api

import (
	"encoding/json"
	"strconv"
	"sync"

	"t20engine/catalog"
)

// AS REGRAS DE ATIVAR UM PODER (ALE-272, fatia 8).
//
// Elas vinham do `power-rules.ts`, que era o metade-regra do antigo hook de
// React: decidir se um poder pode ser usado agora, qual limite o prende, e
// quanto custa entrar numa postura de degraus.
//
// # O que é COBRADO e o que é só crachá
//
// "1/cena" e "1/dia" são cobrados — eles têm contador no banco desde a ALE-222.
// Um "3/dia" e um "1/rodada" saem como crachá e nada mais, e isso é decisão
// registrada, não esquecimento: a mesa conta rodadas, a ficha não. Decisão do
// dono, ALE-272 fatia 8: fica como está.

// activationOfBook é a entrada do registro de ativações.
//
// Ela cresceu nesta fatia: a fatia 5 lia id, nome, tipo, PM e página para achar
// as posturas; a lista de jogo precisa também da AÇÃO que o uso consome, do
// limite, da flag que o gatilho exige e da escala da postura.
type activationOfBook struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Action string `json:"action"`
	// PmCost é CRU porque o catálogo escreve duas coisas nele: um número, ou a
	// palavra "variavel" — 33 das 411 ativações são variáveis. Tipar como `int`
	// não estoura: o `json.Unmarshal` de uma lista guarda o erro de tipo daquele
	// campo, segue em frente, e deixa ZERO no lugar. Medido na bancada: a
	// Paródia do bardo aparecia como "REAÇÃO · 0 PM" com o botão ATIVO, e usá-la
	// não cobrava nada — o oposto do que a regra manda, que é a ficha não decidir
	// um custo que a mesa negocia.
	PmCost json.RawMessage `json:"pmCost"`
	// Uses é `null`, "cena", "dia" ou um número — por isso ele é cru: os três
	// significam coisas diferentes e só dois são cobrados.
	Uses         json.RawMessage   `json:"uses"`
	RequiresFlag string            `json:"requiresFlag"`
	Scaling      *escalaDaAtivacao `json:"scaling"`
	Grant        *concessaoDoPoder `json:"grant"`
	BookPage     int               `json:"bookPage"`
}

// escalaDaAtivacao é a postura que sobe de degrau com o nível.
type escalaDaAtivacao struct {
	BasePm          int    `json:"basePm"`
	StepPm          int    `json:"stepPm"`
	StepLabel       string `json:"stepLabel"`
	FirstStepLevel  int    `json:"firstStepLevel"`
	StepEveryLevels int    `json:"stepEveryLevels"`
}

// concessaoDoPoder é o que a ativação APLICA na ficha — o único caso hoje é a
// reserva de PV temporários da Alma de Bronze.
type concessaoDoPoder struct {
	Kind      string `json:"kind"`
	Scope     string `json:"scope"`
	Attribute string `json:"attribute"`
}

var (
	ativacoesUmaVez  sync.Once
	ativacoesPorID   map[string]activationOfBook
	ativacoesPorNome map[string]activationOfBook
)

func carregaAsAtivacoes() {
	ativacoesUmaVez.Do(func() {
		ativacoesPorID = map[string]activationOfBook{}
		ativacoesPorNome = map[string]activationOfBook{}
		for _, a := range activationsOfBook() {
			ativacoesPorID[a.ID] = a
			ativacoesPorNome[a.Name] = a
		}
	})
}

func activationsOfBook() []activationOfBook {
	bruto, ok := catalog.Resource("activations")
	if !ok {
		return nil
	}
	var lista []activationOfBook
	_ = json.Unmarshal(bruto, &lista)
	return lista
}

// aAtivacaoDe acha a ativação de um poder pelo ID e, se falhar, pelo NOME.
//
// A queda para o nome não é preguiça: os poderes de classe seguem a convenção
// `class.<classe>.<slug>` e casam por id, mas as habilidades de raça e os
// benefícios de origem têm ids próprios (`humano-versatil`,
// `origin-acolito-...`) que o registro de ativações não usa.
//
// E há o caso dos DEGRAUS: "Inspiração +1" e "Fúria +3" são linhas de mesma
// postura, e o nome delas traz o sufixo do degrau. Sem tirar o sufixo, cada
// degrau cairia como passiva silenciosa em vez de resolver para a postura.
func aAtivacaoDe(id, nome string) *activationOfBook {
	carregaAsAtivacoes()
	if spec, tem := ativacoesPorID[id]; tem && id != "" {
		return &spec
	}
	if spec, tem := ativacoesPorNome[nome]; tem {
		return &spec
	}
	if semDegrau := semOSufixoDoDegrau(nome); semDegrau != nome {
		if spec, tem := ativacoesPorNome[semDegrau]; tem {
			return &spec
		}
	}
	return nil
}

// semOSufixoDoDegrau tira o " +N" do fim de "Inspiração +2".
func semOSufixoDoDegrau(nome string) string {
	if len(nome) < 4 {
		return nome
	}
	fim := len(nome)
	for fim > 0 && nome[fim-1] >= '0' && nome[fim-1] <= '9' {
		fim--
	}
	if fim == len(nome) || fim < 2 || nome[fim-1] != '+' || nome[fim-2] != ' ' {
		return nome
	}
	return nome[:fim-2]
}

// ── o LIMITE de usos ─────────────────────────────────────────────────────────

// oEscopoCobrado é "scene", "day" ou "" — e "" quer dizer que o limite existe
// no livro e a ficha NÃO o cobra.
func oEscopoCobrado(spec activationOfBook) string {
	switch string(spec.Uses) {
	case `"cena"`:
		return "scene"
	case `"dia"`:
		return "day"
	}
	return ""
}

// oCrachaDoLimite é o que a tela escreve do limite: "1/cena", "3/dia", ou "".
func oCrachaDoLimite(spec activationOfBook) string {
	cru := string(spec.Uses)
	switch cru {
	case "", "null":
		return ""
	case `"cena"`:
		return "1/cena"
	case `"dia"`:
		return "1/dia"
	case `"rodada"`:
		return "1/rodada"
	}
	var numero int
	if json.Unmarshal(spec.Uses, &numero) == nil {
		return strconv.Itoa(numero) + "/dia"
	}
	return ""
}

// ehCustoVariavel diz se o PM do poder não é um número — "PM variável" na tela,
// e o servidor recusa cobrar por ele: quem sabe o total é a mesa.
func ehCustoVariavel(spec activationOfBook) bool {
	return oPmDaAtivacao(spec) < 0
}

// oPmDaAtivacao é o custo em PM, ou -1 quando ele é variável.
//
// Menos um e não zero: zero é um custo LEGÍTIMO (a maioria das passivas), e
// confundir os dois é o que faria um poder de graça ser tratado como negociado
// com a mesa.
func oPmDaAtivacao(spec activationOfBook) int {
	var numero int
	if json.Unmarshal(spec.PmCost, &numero) == nil {
		return numero
	}
	return -1
}

// ── a DECISÃO de usar ────────────────────────────────────────────────────────

// aDecisaoDoUso responde se o poder pode ser usado AGORA, e por que não.
//
// A ordem das recusas é a da tela antiga, e ela importa: a razão mostrada é a
// PRIMEIRA que barra, então "requer Fúria" aparece antes de "PM insuficiente"
// num poder que precisa das duas coisas — e é a que a pessoa pode resolver
// primeiro.
func aDecisaoDoUso(spec activationOfBook, contexto contextoDoUso) (bool, string) {
	if ehCustoVariavel(spec) {
		return false, "custo variável"
	}
	if spec.RequiresFlag != "" && !contexto.Flags[spec.RequiresFlag] {
		return false, "requer " + spec.RequiresFlag
	}
	switch oEscopoCobrado(spec) {
	case "scene":
		if contexto.UsadoNaCena >= 1 {
			return false, "limite por cena atingido"
		}
	case "day":
		if contexto.UsadoNoDia >= 1 {
			return false, "limite por dia atingido"
		}
	}
	if oPmDaAtivacao(spec) > contexto.PmAtual {
		return false, "PM insuficiente"
	}
	return true, ""
}

// contextoDoUso é o que a decisão precisa saber da ficha AGORA.
type contextoDoUso struct {
	PmAtual     int
	UsadoNaCena int
	UsadoNoDia  int
	Flags       map[string]bool
}

// ── a POSTURA de degraus ─────────────────────────────────────────────────────

// osDegrausDoNivel são os degraus EXTRAS que o nível na classe concede.
//
// O nível é o da CLASSE e não o do personagem (p40): um bárbaro 5/ladino 5 tem
// a Fúria de um bárbaro de nível 5, e não a de um personagem de nível 10.
func osDegrausDoNivel(escala escalaDaAtivacao, nivelNaClasse int) int {
	if escala.StepEveryLevels <= 0 || nivelNaClasse < escala.FirstStepLevel {
		return 0
	}
	return 1 + (nivelNaClasse-escala.FirstStepLevel)/escala.StepEveryLevels
}

// oCustoDaPostura é o que entrar custa com os degraus escolhidos.
func oCustoDaPostura(spec activationOfBook, degraus int) int {
	if spec.Scaling == nil {
		return oPmDaAtivacao(spec)
	}
	return spec.Scaling.BasePm + degraus*spec.Scaling.StepPm
}

// aDecisaoDaPostura responde se dá para entrar na postura com esses degraus.
func aDecisaoDaPostura(spec activationOfBook, degraus, maximo, pmAtual int) (bool, string) {
	if degraus < 0 || degraus > maximo {
		return false, "o nível permite até " + strconv.Itoa(maximo) + " degraus"
	}
	if custo := oCustoDaPostura(spec, degraus); custo > pmAtual {
		return false, "PM insuficiente"
	}
	return true, ""
}
