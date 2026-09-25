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
	Page   int
	// Kind é `instant`, `stance`, `passive`, `triggered-passive` ou "" para o
	// poder que não tem entrada no registro de ativações.
	Kind  string
	Glyph string
	// Cost é "LIVRE · 1 PM" — a ação que o uso consome e o que ele custa.
	Cost string
	// Limit é o crachá do limite: "1/cena", "3/dia". Cobrado só nos dois
	// primeiros; ver `book.ChargedScope`.
	Limit string
	// Spent é "usado 1/1 cena", e só existe para o limite que a ficha cobra.
	Spent string
	// Can e Why são a decisão de usar AGORA e a razão da recusa.
	Can bool
	Why string
	// Stance é o estado da postura, quando a linha é uma.
	Stance  *stanceState
	Command string
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
	context := book.UseContext{CurrentPM: int(dto.MpCurrent), Flags: s.activeFlags(dto)}
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
	uses map[string]character.PowerUse, stances map[string]bool,
) powerRow {
	row := powerRow{
		ID: power.ID, Name: power.Name, Source: shortSource(power.Source),
		Detail: power.Detail, Page: power.Page, Glyph: "BookOpen", Command: power.ID,
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
	row.Cost = writtenCost(*spec)
	context.UsedThisScene, context.UsedToday = uses[spec.ID].Scene, uses[spec.ID].Day
	if scope := book.ChargedScope(*spec); scope != "" {
		row.Spent = writtenSpent(scope, uses[spec.ID])
	}
	row.Can, row.Why = book.UseDecision(*spec, context)
	if spec.Kind == "stance" {
		row.Stance = stanceStateFor(dto, *spec, stances, context)
	}
	return row
}

// stanceStateFor resolve a flag, os degraus do nível e se ela está em curso.
func stanceStateFor(
	dto sheet.CharacterDTO, spec book.Activation, stances map[string]bool, context book.UseContext,
) *stanceState {
	flag := stanceFlag(spec)
	if flag == "" {
		return nil
	}
	state := &stanceState{Flag: flag, Active: stances[flag], BasePm: book.ActivationPm(spec)}
	if spec.Scaling != nil {
		state.BasePm = spec.Scaling.BasePm
		state.StepPm = spec.Scaling.StepPm
		state.StepLabel = spec.Scaling.StepLabel
		state.MaxSteps = book.LevelSteps(*spec.Scaling, character.ClassPowerLevel(dto, spec.ID))
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

// classPowerLevel é o nível NA CLASSE que concede o poder, e não o do
// personagem (p40): um bárbaro 5/ladino 5 tem a Fúria de um bárbaro 5.
//
// A classe sai do id da ativação (`class.barbaro.furia`), que é a convenção do
// catálogo. Sem casar, o nível é o do personagem — o que é generoso, e é a
// escolha certa entre errar para menos e errar para mais numa tela que só
// OFERECE degraus: quem paga é o servidor, que cobra pelo que foi escolhido.
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
// condicional é um encadeado que o motor monta (`engine.ConditionalID`). A flag
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
		if c.Flag != "" && on[engine.ConditionalID(c)] {
			outside[c.Flag] = true
		}
	}
	return outside
}

// paidStances são as posturas com pagamento registrado.
func paidStances(dto sheet.CharacterDTO) map[string]bool {
	outside := map[string]bool{}
	for _, p := range dto.Stances {
		outside[p.Flag] = true
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

// ── o que a TELA escreve ─────────────────────────────────────────────────────

// writtenActions é a economia de ações do livro (p233), em caixa alta porque
// ela é crachá.
var writtenActions = map[string]string{
	"padrao": "PADRÃO", "movimento": "MOVIMENTO", "livre": "LIVRE", "reacao": "REAÇÃO",
	"gratuita": "GRATUITA", "completa": "COMPLETA", "passivo": "PASSIVA", "varia": "VARIA",
}

// writtenCost é "LIVRE · 1 PM" — a ação que o uso consome e o preço.
//
// A POSTURA escreve outra coisa: "POSTURA · 2+ PM". A economia de ação dela
// importa menos que o fato de ser postura (ela DURA), e o "+" avisa que o preço
// sobe com os degraus antes de a pessoa abrir o contador.
func writtenCost(spec book.Activation) string {
	if spec.Kind == "stance" {
		return "POSTURA · " + strconv.Itoa(book.StanceCost(spec, 0)) + stepsMore(spec) + " PM"
	}
	action, found := writtenActions[spec.Action]
	if !found {
		action = strings.ToUpper(spec.Action)
	}
	if book.CostIsVariable(spec) {
		return action + " · PM variável"
	}
	return action + " · " + strconv.Itoa(book.ActivationPm(spec)) + " PM"
}

func stepsMore(spec book.Activation) string {
	if spec.Scaling != nil && spec.Scaling.StepPm > 0 {
		return "+"
	}
	return ""
}

// writtenSpent é "usado 1/1 cena" — o que já se gastou do limite cobrado.
func writtenSpent(scope string, use character.PowerUse) string {
	spent, word := use.Day, "dia"
	if scope == "scene" {
		spent, word = use.Scene, "cena"
	}
	return "usado " + strconv.Itoa(spent) + "/1 " + word
}

// shortSource encurta a procedência para caber no crachá da linha.
//
// "Classe · Bárbaro" vira "Bárbaro" porque a palavra "Classe" é a que se repete
// em quase toda linha — o que distingue é o nome da classe.
func shortSource(source string) string {
	if name, found := strings.CutPrefix(source, "Classe · "); found {
		return name
	}
	if strings.HasPrefix(source, "Raça") {
		return "Raça"
	}
	if strings.HasPrefix(source, "Origem") {
		return "Origem"
	}
	if source == "Poder da Tormenta" {
		return "Tormenta"
	}
	return "Geral"
}

// activationGlyph escolhe o desenho pelo tipo da ativação.
func activationGlyph(kind string) string {
	switch kind {
	case "instant":
		return "Zap"
	case "stance":
		return "Flame"
	case "triggered-passive":
		return "Sparkles"
	}
	return "BookOpen"
}

// writtenPowers é "26 poderes", com o singular certo.
func writtenPowers(n int) string {
	if n == 1 {
		return "1 poder"
	}
	return strconv.Itoa(n) + " poderes"
}

// noActionsPhrase explica a seção vazia, e ela DEPENDE de quem lê: mandar às
// Magias quem não conjura seria mandá-lo a uma aba vazia.
func noActionsPhrase(casts bool) string {
	if casts {
		return "Nenhuma ação ativável — suas magias estão na aba Magias."
	}
	return "Nenhuma ação ativável. Suas habilidades são passivas."
}

// powerBadge é a classe do crachá — a postura sai na tinta arcana, que é a
// mesma da tripla mágica do Combate.
func powerBadge(kind string) string {
	base := "shrink-0 rounded-full border px-1.5 py-px text-3xs font-semibold uppercase tracking-wide"
	if kind == "stance" {
		return base + " border-arcane/40 text-arcane-ink"
	}
	return base + " border-grimorio-iron text-muted-foreground"
}

// costStancePreview soma o custo dos degraus na tela.
//
// PRÉVIA e não decisão: quem cobra é o servidor, com o teto de degraus do nível
// e o PM disponível. Escrever a regra aqui daria uma segunda conta do mesmo
// número.
func costStancePreview(row powerRow) string {
	base := strconv.Itoa(row.Stance.BasePm)
	step := strconv.Itoa(row.Stance.StepPm)
	return "(" + base + " + " + step + " * $stance_degrees) + ' PM'"
}
