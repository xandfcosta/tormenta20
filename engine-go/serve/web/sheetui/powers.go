package sheetui

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"t20engine/app/character"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/search"
	"t20engine/domain/sheet"
)

// A aba PODERES como dado.
//
// UMA lista, a da mesa, e nenhum modo de edição: escolher poder acontece uma vez
// por nível e mora num DIÁLOGO. O que fica na tela é o que a mesa usa.
//
// AÇÕES são o que se ativa: instantâneos e posturas. Elas vêm ordenadas com a
// postura ATIVA primeiro — é a que a pessoa vai querer encerrar — e depois por
// PM crescente, com o custo variável por último.
//
// PASSIVAS ficam recolhidas atrás de "mostrar (N)": elas já entram nos números
// da ficha, então na mesa são referência e não ação. A exceção que aparece com o
// bloco fechado é a passiva de GATILHO com o gatilho no ar — ela está fazendo
// efeito AGORA e sumir seria mentir.

// powersPanel é a aba Poderes pronta para desenhar.
type powersPanel struct {
	Actions  []powerRow
	Passives []powerRow
	// LiveTriggers são as passivas de gatilho com o gatilho ATIVO, para a linha
	// que aparece com o bloco recolhido.
	LiveTriggers []powerRow
	// Results é a busca plana por nome; vazia quando ninguém digitou nada.
	Results []powerRow
	Search  string
	// Total é quantos poderes o personagem tem, antes de qualquer filtro.
	Total int
	// IsCaster decide a frase de "nenhuma ação": quem conjura é mandado para a
	// aba Magias, e quem não conjura não é — mandá-lo a uma aba vazia seria pior
	// que não dizer nada.
	IsCaster bool
}

// powerRow é um poder na tela.
type powerRow struct {
	ID     string
	Name   string
	Source string
	Detail string
	// Requires é o pré-requisito por extenso, vazio quando não há.
	Requires string
	Page     int
	// Kind é `instant`, `stance`, `passive`, `triggered-passive` ou "" para o
	// poder que não tem entrada no registro de ativações.
	Kind  string
	Glyph string
	// Cost é "LIVRE · 1 PM" — a ação que o uso consome e o que ele custa.
	Cost string
	// Action é a chave CRUA do catálogo (`padrao`, `livre`, `reacao`…), e ela
	// existe para quem AGRUPA: a superfície de Ações põe cada linha no custo que
	// ela paga, e ler isso do `Cost` já escrito seria um analisador de texto de
	// tela — o rótulo muda, o grupo muda junto, e em silêncio.
	Action string
	// Limit é o crachá do limite: "1/cena", "3/dia". Cobrado só nos dois
	// primeiros; ver `book.ChargedScope`.
	Limit string
	// Spent é "usado 1/1 cena", e só existe para o limite que a ficha cobra.
	Spent string
	// Can e Why são a decisão de usar AGORA e a razão da recusa.
	Can bool
	Why string
	// Stance é o estado da postura, quando a linha é uma.
	Stance *stanceState
	// Degrees são os degraus de USO do instantâneo que os vende (a Marca da
	// Presa), e InEffect o que a concessão dele dá agora ("+1d12"), quando em
	// curso. Ver `powers_degrees.go`.
	Degrees  *useDegrees
	InEffect string
	Command  string
}

// stanceState é o que a tela precisa saber de uma postura.
type stanceState struct {
	Flag string
	// Active diz se ela está EM CURSO — o que troca o botão de Ativar para
	// Encerrar.
	Active bool
	// MaxSteps são os degraus que o nível na classe concede. Zero quer dizer
	// custo fixo, e aí a ativação é de um toque só, sem diálogo.
	MaxSteps int
	BasePm   int
	StepPm   int
	// StepLabel é o que cada degrau compra ("+1 no bônus de Fúria").
	StepLabel string
	// BaseBonus é o bônus do degrau ZERO — o +2 da Fúria, o +1 da Inspiração —,
	// e é com ele que o contador fala a língua do livro. Zero na postura que não
	// move número; ver `book.StanceBaseBonus`.
	BaseBonus int
	// Bonus é o que a postura está dando AGORA: a base mais os degraus pagos.
	// Só faz sentido com ela em curso, e é o número pelo qual a mesa a chama.
	Bonus int
	// BonusNote avisa que um poder vai MULTIPLICAR este bônus depois da conta —
	// hoje só a Fúria Titânica (p42). Vazio para quem não tem; ver
	// `stanceBonusNote`.
	BonusNote string
}

// powersPanelOf monta a aba.
func (s Scene) powersPanelOf(dto sheet.CharacterDTO, query string) powersPanel {
	panel := powersPanel{Search: query, IsCaster: len(casterClassesOf(dto)) > 0}
	rows := s.powerRowsOf(dto)
	panel.Total = len(rows)
	if term := search.Fold(strings.TrimSpace(query)); term != "" {
		panel.Results = filtradasPorNome(rows, term)
		return panel
	}
	panel.Actions, panel.Passives = separadasPorUso(rows)
	panel.LiveTriggers = airTriggers(panel.Passives)
	return panel
}

// powerRowsOf traduz o acervo em linhas de tela, resolvendo a ativação de cada
// poder e o estado de jogo dele.
func (s Scene) powerRowsOf(dto sheet.CharacterDTO) []powerRow {
	context := book.UseContext{CurrentPM: int(dto.ManaAvailable()), Flags: s.activeFlags(dto)}
	uses := character.PowerUses(dto)
	stances := paidStances(dto)
	rows := []powerRow{}
	for _, power := range ownedPowersOf(dto) {
		rows = append(rows, powerRowFor(dto, power, context, uses, stances))
	}
	return stanceRepeatedSem(rows)
}

func powerRowFor(
	dto sheet.CharacterDTO, power ownedPower, context book.UseContext,
	uses map[string]character.PowerUse, stances map[string]int,
) powerRow {
	row := powerRow{
		ID: power.ID, Name: power.Name, Source: shortSource(power.Source),
		Detail: power.Detail, Requires: book.PowerRequirements(power.ID), Page: power.Page, Glyph: "BookOpen", Command: power.ID,
	}
	spec := book.ActivationOf(power.ID, power.Name)
	if spec == nil {
		return row
	}
	row.ID, row.Command = spec.ID, spec.ID
	row.Kind, row.Glyph = spec.Kind, activationGlyph(spec.Kind)
	// O NOME DA AÇÃO é o da ATIVAÇÃO, e não o da linha do catálogo.
	//
	// "Inspiração +1" a "+5" são cinco linhas de classe e UMA postura na mesa: o
	// número no fim é o degrau, e ele é justamente o que se escolhe ao entrar.
	// Manter o sufixo daria um botão "Ativar Inspiração +1" que ativa qualquer
	// degrau — a tela prometeria uma escolha que o gesto não faz.
	if spec.Kind == "stance" || spec.Kind == "instant" {
		row.Name = spec.Name
	}
	if spec.BookPage > 0 {
		row.Page = spec.BookPage
	}
	row.Limit = limitBadge(*spec)
	// A POSTURA É RESOLVIDA ANTES DO PREÇO, e a ordem é a correção: o `+` do
	// "2+ PM" é sobre ESTE personagem, e quem sabe se ele tem degrau a comprar é
	// o estado da postura.
	if spec.Kind == "stance" {
		row.Stance = stanceStateFor(dto, *spec, stances, context)
	}
	row.Degrees = useDegreesFor(dto, *spec)
	row.InEffect = grantInEffect(dto, spec.ID)
	row.Cost, row.Action = writtenCost(*spec, row.Stance), spec.Action
	context.UsedThisScene, context.UsedToday = uses[spec.ID].Scene, uses[spec.ID].Day
	if scope := book.ChargedScope(*spec); scope != "" {
		row.Spent = writtenSpent(scope, uses[spec.ID])
	}
	row.Can, row.Why = book.UseDecision(*spec, context)
	return row
}

// stanceStateFor resolve a flag, os degraus do nível e se ela está em curso.
func stanceStateFor(
	dto sheet.CharacterDTO, spec book.Activation, stances map[string]int, context book.UseContext,
) *stanceState {
	flag := stanceFlag(spec)
	if flag == "" {
		return nil
	}
	steps, active := stances[flag]
	state := &stanceState{
		Flag: flag, Active: active, Bonus: book.StanceBonusInEffect(flag, steps),
		BasePm: book.ActivationPm(spec), BaseBonus: book.StanceBaseBonus(flag),
		BonusNote: stanceBonusNote(dto, spec.ID),
	}
	if spec.Scaling != nil {
		state.BasePm = spec.Scaling.BasePm
		state.StepPm = spec.Scaling.StepPm
		state.StepLabel = spec.Scaling.StepLabel
		state.MaxSteps = character.PowerMaxSteps(dto, spec)
	}
	return state
}

// stanceFlag acha a flag que a postura acende.
//
// Ela sai do CATÁLOGO — a postura não declara a própria flag, e derivá-la do id
// acertaria as duas de hoje e erraria calado na terceira.
func stanceFlag(spec book.Activation) string {
	for flag, stance := range book.StancesFromCatalog() {
		if stance.Name == spec.Name {
			return flag
		}
	}
	return ""
}

// limitBadge é o que a TELA escreve do limite: "1/cena", "3/dia", ou "".
//
// Ele ficou na cena quando as regras de ativação subiram para o `domain/book`
// (ALE-351), e a linha entre os dois é a mesma do guia: o `book.ChargedScope`
// decide o que a ficha COBRA, este escreve o que a pessoa LÊ. Um "3/dia" sai
// como crachá e não é cobrado — a mesma entrada, duas respostas.
func limitBadge(spec book.Activation) string {
	raw := string(spec.Uses)
	switch raw {
	case "", "null":
		return ""
	case `"cena"`:
		return "1/cena"
	case `"dia"`:
		return "1/dia"
	case `"rodada"`:
		return "1/rodada"
	}
	var number int
	if json.Unmarshal(spec.Uses, &number) == nil {
		return strconv.Itoa(number) + "/dia"
	}
	return ""
}

// activeFlags são as FLAGS levantadas agora, e elas não estão no banco.
//
// O que o banco guarda é a lista de condicionais LIGADOS, e o id de um
// condicional é um encadeado que o motor monta (`engine.TermID`). A flag
// mora do outro lado: é o motor quem diz, para cada condicional oferecido, qual
// flag ele acende. Então a pergunta "a Fúria está em pé?" é uma junção entre o
// que o jogador ligou e o que o motor oferece — e é por isso que ela não é uma
// consulta.
//
// Sem catálogo primado não há condicional oferecido, e o mapa sai vazio: a
// consequência é a tela não OFERECER o poder de gatilho, que é o lado seguro.
func (s Scene) activeFlags(dto sheet.CharacterDTO) map[string]bool {
	outside := map[string]bool{}
	if dto.Ruleset == nil {
		return outside
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return outside
	}
	on := sheet.ToStringSet(dto.Conditionals)
	for _, c := range engine.ComputeItemEffects(dto.Ruleset.ActiveItemsFor(ec)).Conditional {
		if c.Flag != "" && on[engine.FlagGroupID(c.Flag)] {
			outside[c.Flag] = true
		}
	}
	return outside
}

// paidStances são as posturas com pagamento registrado, e com QUANTOS DEGRAUS
// cada uma pagou.
//
// Os degraus vêm junto porque a tela diz o bônus EM CURSO, e ele é base mais
// degraus: a presença da flag responde "está em pé?", e o valor responde "em
// pé valendo quanto?". Um `map[string]bool` responderia só a primeira.
func paidStances(dto sheet.CharacterDTO) map[string]int {
	outside := map[string]int{}
	for _, p := range dto.Stances {
		outside[p.Flag] = int(p.Steps)
	}
	return outside
}

// stanceRepeatedSem junta os DEGRAUS numa linha só.
//
// "Inspiração +1" até "+5" são cinco linhas do catálogo e UMA postura na mesa —
// mostrar as cinco daria cinco botões de Ativar para a mesma coisa.
func stanceRepeatedSem(rows []powerRow) []powerRow {
	seen := map[string]bool{}
	outside := []powerRow{}
	for _, row := range rows {
		if row.Kind == "stance" || row.Kind == "instant" {
			if seen[row.ID] {
				continue
			}
			seen[row.ID] = true
		}
		outside = append(outside, row)
	}
	return outside
}

// separadasPorUso parte as linhas nas duas seções e ordena as ações.
func separadasPorUso(rows []powerRow) (actions, passives []powerRow) {
	for _, row := range rows {
		if row.Kind == "instant" || row.Kind == "stance" {
			actions = append(actions, row)
			continue
		}
		passives = append(passives, row)
	}
	sort.SliceStable(actions, func(a, b int) bool {
		if activeOne(actions[a]) != activeOne(actions[b]) {
			return activeOne(actions[a])
		}
		return rowPm(actions[a]) < rowPm(actions[b])
	})
	return actions, passives
}

func activeOne(row powerRow) bool {
	return row.Stance != nil && row.Stance.Active
}

// rowPm é o custo para ordenar. O custo variável vai para o FIM, e é por
// isso que ele vira um número grande em vez de ganhar um caso próprio.
func rowPm(row powerRow) int {
	if strings.Contains(row.Cost, "variável") {
		return 999
	}
	for _, chunk := range strings.Fields(row.Cost) {
		if n, err := strconv.Atoi(chunk); err == nil {
			return n
		}
	}
	return 0
}

// airTriggers são as passivas de gatilho que estão fazendo efeito agora.
func airTriggers(passives []powerRow) []powerRow {
	outside := []powerRow{}
	for _, row := range passives {
		if row.Kind == "triggered-passive" && row.Can {
			outside = append(outside, row)
		}
	}
	return outside
}

func filtradasPorNome(rows []powerRow, term string) []powerRow {
	outside := []powerRow{}
	for _, row := range rows {
		if strings.Contains(search.Fold(row.Name), term) {
			outside = append(outside, row)
		}
	}
	return outside
}
