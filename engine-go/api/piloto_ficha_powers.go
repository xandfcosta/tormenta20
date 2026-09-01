package api

import (
	"sort"
	"strconv"
	"strings"

	"t20engine/engine"
	"t20engine/sheet"
)

// A aba PODERES como dado (ALE-272, fatia 8).
//
// UMA lista, a da mesa. A ficha antiga tinha dois MODOS — jogo e edição — e o
// dono resumiu o resultado em "está difícil de ser usada": o modo de edição
// abria sozinho sempre que havia pendência (o estado normal de quem subiu de
// nível) e o cromo dele comia 44% do painel no telefone (ALE-217). Escolher
// poder acontece uma vez por nível e virou diálogo; o que fica na tela é o que
// a mesa usa.
//
// # Duas seções, e a ordem das AÇÕES é a regra
//
// AÇÕES são o que se ativa: instantâneos e posturas. Elas vêm ordenadas com a
// postura ATIVA primeiro — é a que a pessoa vai querer encerrar — e depois por
// PM crescente, com o custo variável por último.
//
// PASSIVAS ficam recolhidas atrás de "mostrar (N)": elas já entram nos números
// da ficha, então na mesa são referência e não ação. A exceção que aparece com
// o bloco fechado é a passiva de GATILHO com o gatilho no ar — ela está fazendo
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
	// primeiros; ver `oEscopoCobrado`.
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
func (s *Server) powersPanelOf(dto sheet.CharacterDTO, busca string) powersPanel {
	panel := powersPanel{Search: busca, IsCaster: len(casterClassesOf(dto)) > 0}
	linhas := s.powerRowsOf(dto)
	panel.Total = len(linhas)
	if termo := foldAccents(strings.TrimSpace(busca)); termo != "" {
		panel.Results = filtradasPorNome(linhas, termo)
		return panel
	}
	panel.Actions, panel.Passives = separadasPorUso(linhas)
	panel.LiveTriggers = osGatilhosNoAr(panel.Passives)
	return panel
}

// powerRowsOf traduz o acervo em linhas de tela, resolvendo a ativação de cada
// poder e o estado de jogo dele.
func (s *Server) powerRowsOf(dto sheet.CharacterDTO) []powerRow {
	contexto := contextoDoUso{PmAtual: int(dto.MpCurrent), Flags: s.asFlagsAtivas(dto)}
	usos := osUsosPorPoder(dto)
	posturas := asPosturasPagas(dto)
	linhas := []powerRow{}
	for _, poder := range ownedPowersOf(dto) {
		linhas = append(linhas, aLinhaDoPoder(dto, poder, contexto, usos, posturas))
	}
	return semPosturasRepetidas(linhas)
}

func aLinhaDoPoder(
	dto sheet.CharacterDTO, poder ownedPower, contexto contextoDoUso,
	usos map[string]usoDoPoder, posturas map[string]bool,
) powerRow {
	linha := powerRow{
		ID: poder.ID, Name: poder.Name, Source: aFonteCurta(poder.Source),
		Detail: poder.Detail, Page: poder.Page, Glyph: "BookOpen", Command: poder.ID,
	}
	spec := aAtivacaoDe(poder.ID, poder.Name)
	if spec == nil {
		return linha
	}
	linha.ID, linha.Command = spec.ID, spec.ID
	linha.Kind, linha.Glyph = spec.Kind, oGlifoDaAtivacao(spec.Kind)
	// O NOME DA AÇÃO é o da ATIVAÇÃO, e não o da linha do catálogo.
	//
	// "Inspiração +1" a "+5" são cinco linhas de classe e UMA postura na mesa: o
	// número no fim é o degrau, e ele é justamente o que se escolhe ao entrar.
	// Manter o sufixo daria um botão "Ativar Inspiração +1" que ativa qualquer
	// degrau — a tela prometeria uma escolha que o gesto não faz.
	if spec.Kind == "stance" || spec.Kind == "instant" {
		linha.Name = spec.Name
	}
	if spec.BookPage > 0 {
		linha.Page = spec.BookPage
	}
	linha.Limit = oCrachaDoLimite(*spec)
	linha.Cost = oCustoEscrito(*spec)
	contexto.UsadoNaCena, contexto.UsadoNoDia = usos[spec.ID].Cena, usos[spec.ID].Dia
	if escopo := oEscopoCobrado(*spec); escopo != "" {
		linha.Spent = oGastoEscrito(escopo, usos[spec.ID])
	}
	linha.Can, linha.Why = aDecisaoDoUso(*spec, contexto)
	if spec.Kind == "stance" {
		linha.Stance = oEstadoDaPostura(dto, *spec, posturas, contexto)
	}
	return linha
}

// oEstadoDaPostura resolve a flag, os degraus do nível e se ela está em curso.
func oEstadoDaPostura(
	dto sheet.CharacterDTO, spec activationOfBook, posturas map[string]bool, contexto contextoDoUso,
) *stanceState {
	flag := aFlagDaPostura(spec)
	if flag == "" {
		return nil
	}
	estado := &stanceState{Flag: flag, Active: posturas[flag], BasePm: oPmDaAtivacao(spec)}
	if spec.Scaling != nil {
		estado.BasePm = spec.Scaling.BasePm
		estado.StepPm = spec.Scaling.StepPm
		estado.StepLabel = spec.Scaling.StepLabel
		estado.MaxSteps = osDegrausDoNivel(*spec.Scaling, oNivelNaClasseDoPoder(dto, spec.ID))
	}
	return estado
}

// aFlagDaPostura acha a flag que a postura acende.
//
// Ela sai do mapa que a fatia 5 montou lendo o catálogo — a postura NÃO declara
// a própria flag, e derivá-la do id acertaria as duas de hoje e erraria calado
// na terceira.
func aFlagDaPostura(spec activationOfBook) string {
	for flag, postura := range stancesFromCatalog() {
		if postura.Name == spec.Name {
			return flag
		}
	}
	return ""
}

// oNivelNaClasseDoPoder é o nível NA CLASSE que concede o poder, e não o do
// personagem (p40): um bárbaro 5/ladino 5 tem a Fúria de um bárbaro 5.
//
// A classe sai do id da ativação (`class.barbaro.furia`), que é a convenção do
// catálogo. Sem casar, o nível é o do personagem — o que é generoso, e é a
// escolha certa entre errar para menos e errar para mais numa tela que só
// OFERECE degraus: quem paga é o servidor, que cobra pelo que foi escolhido.
func oNivelNaClasseDoPoder(dto sheet.CharacterDTO, activationID string) int {
	for _, classe := range dto.Classes {
		if strings.Contains(activationID, "."+foldAccents(strings.ToLower(classe.ClassName))+".") {
			return int(classe.Level)
		}
	}
	return int(dto.Level)
}

// asFlagsAtivas são as FLAGS levantadas agora, e elas não estão no banco.
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
func (s *Server) asFlagsAtivas(dto sheet.CharacterDTO) map[string]bool {
	fora := map[string]bool{}
	if s.catalogs == nil {
		return fora
	}
	ec, err := engineCharacterFrom(dto)
	if err != nil {
		return fora
	}
	ligados := toStringSet(dto.Conditionals)
	for _, c := range engine.ComputeItemEffects(s.catalogs.ActiveItemsFor(ec)).Conditional {
		if c.Flag != "" && ligados[engine.ConditionalID(c)] {
			fora[c.Flag] = true
		}
	}
	return fora
}

// asPosturasPagas são as posturas com pagamento registrado.
func asPosturasPagas(dto sheet.CharacterDTO) map[string]bool {
	fora := map[string]bool{}
	for _, p := range dto.Stances {
		fora[p.Flag] = true
	}
	return fora
}

type usoDoPoder struct{ Cena, Dia int }

func osUsosPorPoder(dto sheet.CharacterDTO) map[string]usoDoPoder {
	fora := map[string]usoDoPoder{}
	for _, u := range dto.PowerUses {
		conta := fora[u.PowerID]
		if u.Scope == "scene" {
			conta.Cena = int(u.Used)
		} else {
			conta.Dia = int(u.Used)
		}
		fora[u.PowerID] = conta
	}
	return fora
}

// semPosturasRepetidas junta os DEGRAUS numa linha só.
//
// "Inspiração +1" até "+5" são cinco linhas do catálogo e UMA postura na mesa —
// mostrar as cinco daria cinco botões de Ativar para a mesma coisa.
func semPosturasRepetidas(linhas []powerRow) []powerRow {
	vistas := map[string]bool{}
	fora := []powerRow{}
	for _, linha := range linhas {
		if linha.Kind == "stance" || linha.Kind == "instant" {
			if vistas[linha.ID] {
				continue
			}
			vistas[linha.ID] = true
		}
		fora = append(fora, linha)
	}
	return fora
}

// separadasPorUso parte as linhas nas duas seções e ordena as ações.
func separadasPorUso(linhas []powerRow) (acoes, passivas []powerRow) {
	for _, linha := range linhas {
		if linha.Kind == "instant" || linha.Kind == "stance" {
			acoes = append(acoes, linha)
			continue
		}
		passivas = append(passivas, linha)
	}
	sort.SliceStable(acoes, func(a, b int) bool {
		if aAtiva(acoes[a]) != aAtiva(acoes[b]) {
			return aAtiva(acoes[a])
		}
		return oPmDaLinha(acoes[a]) < oPmDaLinha(acoes[b])
	})
	return acoes, passivas
}

func aAtiva(linha powerRow) bool {
	return linha.Stance != nil && linha.Stance.Active
}

// oPmDaLinha é o custo para ordenar. O custo variável vai para o FIM, e é por
// isso que ele vira um número grande em vez de ganhar um caso próprio.
func oPmDaLinha(linha powerRow) int {
	if strings.Contains(linha.Cost, "variável") {
		return 999
	}
	for _, pedaco := range strings.Fields(linha.Cost) {
		if n, err := strconv.Atoi(pedaco); err == nil {
			return n
		}
	}
	return 0
}

// osGatilhosNoAr são as passivas de gatilho que estão fazendo efeito agora.
func osGatilhosNoAr(passivas []powerRow) []powerRow {
	fora := []powerRow{}
	for _, linha := range passivas {
		if linha.Kind == "triggered-passive" && linha.Can {
			fora = append(fora, linha)
		}
	}
	return fora
}

func filtradasPorNome(linhas []powerRow, termo string) []powerRow {
	fora := []powerRow{}
	for _, linha := range linhas {
		if strings.Contains(foldAccents(linha.Name), termo) {
			fora = append(fora, linha)
		}
	}
	return fora
}

// ── o que a TELA escreve ─────────────────────────────────────────────────────

// asAcoesEscritas é a economia de ações do livro (p105), em caixa alta porque
// ela é crachá.
var asAcoesEscritas = map[string]string{
	"padrao": "PADRÃO", "movimento": "MOVIMENTO", "livre": "LIVRE", "reacao": "REAÇÃO",
	"gratuita": "GRATUITA", "completa": "COMPLETA", "passivo": "PASSIVA", "varia": "VARIA",
}

// oCustoEscrito é "LIVRE · 1 PM" — a ação que o uso consome e o preço.
//
// A POSTURA escreve outra coisa: "POSTURA · 2+ PM". A economia de ação dela
// importa menos que o fato de ser postura (ela DURA), e o "+" avisa que o preço
// sobe com os degraus antes de a pessoa abrir o contador.
func oCustoEscrito(spec activationOfBook) string {
	if spec.Kind == "stance" {
		return "POSTURA · " + strconv.Itoa(oCustoDaPostura(spec, 0)) + oMaisDosDegraus(spec) + " PM"
	}
	acao, tem := asAcoesEscritas[spec.Action]
	if !tem {
		acao = strings.ToUpper(spec.Action)
	}
	if ehCustoVariavel(spec) {
		return acao + " · PM variável"
	}
	return acao + " · " + strconv.Itoa(oPmDaAtivacao(spec)) + " PM"
}

func oMaisDosDegraus(spec activationOfBook) string {
	if spec.Scaling != nil && spec.Scaling.StepPm > 0 {
		return "+"
	}
	return ""
}

// oGastoEscrito é "usado 1/1 cena" — o que já se gastou do limite cobrado.
func oGastoEscrito(escopo string, uso usoDoPoder) string {
	gasto, palavra := uso.Dia, "dia"
	if escopo == "scene" {
		gasto, palavra = uso.Cena, "cena"
	}
	return "usado " + strconv.Itoa(gasto) + "/1 " + palavra
}

// aFonteCurta encurta a procedência para caber no crachá da linha.
//
// "Classe · Bárbaro" vira "Bárbaro" porque a palavra "Classe" é a que se repete
// em quase toda linha — o que distingue é o nome da classe.
func aFonteCurta(fonte string) string {
	if nome, achou := strings.CutPrefix(fonte, "Classe · "); achou {
		return nome
	}
	if strings.HasPrefix(fonte, "Raça") {
		return "Raça"
	}
	if strings.HasPrefix(fonte, "Origem") {
		return "Origem"
	}
	if fonte == "Poder da Tormenta" {
		return "Tormenta"
	}
	return "Geral"
}

// oGlifoDaAtivacao escolhe o desenho pelo tipo da ativação.
func oGlifoDaAtivacao(kind string) string {
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

// osPoderesEscrito é "26 poderes", com o singular certo.
func osPoderesEscrito(n int) string {
	if n == 1 {
		return "1 poder"
	}
	return strconv.Itoa(n) + " poderes"
}

// aFraseDeSemAcoes explica a seção vazia, e ela DEPENDE de quem lê: mandar às
// Magias quem não conjura seria mandá-lo a uma aba vazia.
func aFraseDeSemAcoes(conjura bool) string {
	if conjura {
		return "Nenhuma ação ativável — suas magias estão na aba Magias."
	}
	return "Nenhuma ação ativável. Suas habilidades são passivas."
}

// oCrachaDoPoder é a classe do crachá — a postura sai na tinta arcana, que é a
// mesma da tripla mágica do Combate.
func oCrachaDoPoder(kind string) string {
	base := "shrink-0 rounded-full border px-1.5 py-px text-3xs font-semibold uppercase tracking-wide"
	if kind == "stance" {
		return base + " border-arcane/40 text-arcane-ink"
	}
	return base + " border-grimorio-iron text-muted-foreground"
}

// aPreviaDoCustoDaPostura soma o custo dos degraus na tela.
//
// PRÉVIA e não decisão: quem cobra é o servidor, com o teto de degraus do nível
// e o PM disponível. Escrever a regra aqui daria uma segunda conta do mesmo
// número — o defeito que a ALE-110 registrou.
func aPreviaDoCustoDaPostura(linha powerRow) string {
	base := strconv.Itoa(linha.Stance.BasePm)
	passo := strconv.Itoa(linha.Stance.StepPm)
	return "(" + base + " + " + passo + " * $poderdegraus) + ' PM'"
}
