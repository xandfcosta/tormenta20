package sheetui

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"sync"

	"t20engine/book"
	"t20engine/catalog"
	"t20engine/engine"
	"t20engine/sheet"
)

// A aba EFEITOS como dado (ALE-272, fatia 5).
//
// É tudo que está mexendo nos números do personagem AGORA, em quatro blocos que
// diferem por QUEM é dono do estado:
//
//  1. CONDIÇÕES do livro (p394-395) — coluna `activeConditions`. Elas MOVEM os
//     números: uma condição que fosse só crachá foi o defeito da ALE-28.
//  2. POSTURAS em curso, com o que cada uma custou. O interruptor de LIGAR mora
//     nos Poderes, onde o PM é cobrado; aqui elas só se leem e se encerram.
//  3. EFEITOS ATIVOS — consumível usado e magia de bônus aplicada, com escopo de
//     cena ou dia.
//  4. SITUAÇÃO — o opt-in de contexto (terreno, tipo de alvo, item caseiro).
//
// # As posturas saem do CATÁLOGO, e a tabela do front era cópia
//
// A SPA guarda um `FLAG_ACTIVATIONS` escrito à mão com as duas posturas, o PM e a
// página. As duas JÁ ESTAVAM no `activations.json` como `"kind": "stance"`, e a
// FLAG de cada uma sai do poder de mesmo id, lendo o `condition.flag` dos
// modificadores dele. Não há tabela nova aqui: há uma leitura do que o catálogo
// já dizia, e o que morre com a SPA é a cópia.

// effectsPanel é a aba Efeitos pronta para desenhar.
type effectsPanel struct {
	Conditions []conditionRow
	// ConditionOptions são as que ainda cabem — o catálogo menos as ativas.
	ConditionOptions []pickerOption
	Stances          []stanceRow
	Applied          []appliedEffectRow
	// BuffOptions são as magias que têm efeito aplicável, para o diálogo.
	BuffOptions []pickerOption
	Situational []situationalRow
	// AlwaysOn são as flags de item equipado que não têm interruptor — a
	// "Fadiga ao dormir" da armadura pesada. Elas aparecem em modo de leitura
	// para o bloco não contradizer o aviso que a mesma flag produz no cabeçalho.
	AlwaysOn []alwaysOnRow
	// Count é o que o crachá da aba mostra, e ele conta COISAS e não modificadores
	// do motor: a Fúria entra como oito modificadores e é UMA coisa na mesa.
	Count int
}

type conditionRow struct {
	ID      string
	Name    string
	Effect  string
	Page    int
	Command string
}

type pickerOption struct {
	ID    string
	Label string
	// Detail é a segunda linha do item — o efeito da condição, o círculo da
	// magia. Sem ela o jogador escolhe por nome decorado.
	Detail  string
	Command string
}

type stanceRow struct {
	Flag string
	Name string
	// Paid é "2 PM" — o que foi cobrado para entrar. Ele fica visível porque
	// encerrar não devolve nada, e a pessoa merece saber o que gastou.
	Paid    string
	Steps   string
	Command string
}

type appliedEffectRow struct {
	ID     int64
	Name   string
	Scope  string
	Detail string
	// Modifiers são as linhas de "o que isto faz", que é o que separa um efeito
	// aplicado de um nome numa lista.
	Modifiers []breakdownRow
	Command   string
}

type situationalRow struct {
	// Key é o que o comando manda: a chave do CONDICIONAL, ou a flag quando o
	// grupo tem uma.
	Key    string
	Label  string
	Source string
	Active bool
	// Folded diz que a linha liga MAIS DE UM modificador de uma vez. Um item
	// caseiro com três modificadores é UM interruptor, senão a pessoa deixa
	// metade do efeito ligado.
	Folded    bool
	Modifiers []breakdownRow
	Command   string
}

type alwaysOnRow struct {
	Label  string
	Source string
}

// ── as posturas, lidas do catálogo ───────────────────────────────────────────

// stanceByFlag é a postura de cada flag: `furia` → Fúria, 2 PM, p40.
type stanceOfBook struct {
	Flag string
	Name string
	PM   int
	Page int
}

var (
	stancesOnce   sync.Once
	stancesByFlag map[string]stanceOfBook
)

// stancesFromCatalog liga as ativações de `kind: "stance"` à flag que elas
// acendem.
//
// A FLAG NÃO É ADIVINHADA do id: ela sai do poder de MESMO id, lendo o
// `condition.flag` dos modificadores dele. Derivar do último pedaço do id
// acertaria as duas de hoje e erraria calado na terceira.
func stancesFromCatalog() map[string]stanceOfBook {
	stancesOnce.Do(func() {
		stancesByFlag = map[string]stanceOfBook{}
		flags := book.ClassPowerFlags()
		for _, a := range book.Activations() {
			if a.Kind != "stance" {
				continue
			}
			flag := flags[a.ID]
			if flag == "" {
				flag = flags[a.ID+stanceStep(flags, a.ID)]
			}
			if flag == "" {
				continue
			}
			stancesByFlag[flag] = stanceOfBook{Flag: flag, Name: a.Name, PM: activationPm(a), Page: a.BookPage}
		}
	})
	return stancesByFlag
}

// degrauDaPostura acha o sufixo do DEGRAU quando a postura não declara a flag no
// poder de id exato.
//
// O catálogo trata as duas posturas de formas DIFERENTES, e isso é achado desta
// fatia: `class.barbaro.furia` carrega os modificadores no poder de id exato,
// enquanto `class.bardo.inspiracao` os põe nos degraus numerados
// (`inspiracao-1`, `-2`, …) e deixa o id base sem modificador nenhum. Ligar só
// pelo id exato achava UMA das duas — e passava calado, porque a Inspiração
// simplesmente não aparecia na lista de posturas.
//
// O sufixo aceito é `-<dígitos>` e MAIS NADA. Um prefixo solto casaria
// `class.barbaro.furia-da-savana`, que é outro poder — e no dia em que ele
// ligasse uma flag, a postura errada herdaria a dele.
func stanceStep(flags map[string]string, base string) string {
	for i := 1; i <= 9; i++ {
		sufixo := "-" + strconv.Itoa(i)
		if flags[base+sufixo] != "" {
			return sufixo
		}
	}
	return ""
}

// ── a montagem ───────────────────────────────────────────────────────────────

// effectsPanelOf computa a aba Efeitos de um personagem.
//
// Ela pergunta ao motor QUAIS condicionais existem — a lista de opt-ins que
// aquela ficha oferece, dado o que ela veste e sabe — e não o que está ligado:
// o que está ligado é o `dto.Conditionals`, que é do jogador. Sem catálogo
// primado a aba mostra o que não depende do motor (condições, posturas e
// efeitos aplicados) e perde a Situação, que é derivada.
func (s Scene) effectsPanelOf(dto sheet.CharacterDTO) effectsPanel {
	if s.deps.Catalogs() == nil {
		return effectsPanelFor(dto, nil, nil)
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return effectsPanelFor(dto, nil, nil)
	}
	oferecidos := engine.ComputeItemEffects(s.deps.Catalogs().ActiveItemsFor(ec)).Conditional
	return effectsPanelFor(dto, oferecidos, s.deps.Catalogs().ComputeEquippedFlags(ec.Items))
}

// effectsPanelFor monta a aba inteira.
func effectsPanelFor(dto sheet.CharacterDTO, offered []engine.ConditionalEffect, flags []engine.EquippedFlag) effectsPanel {
	ativos := sheet.ToStringSet(dto.Conditionals)
	panel := effectsPanel{
		Conditions:       conditionRowsOf(dto),
		ConditionOptions: conditionOptionsFor(dto),
		Stances:          stanceRowsOf(dto),
		Applied:          appliedEffectRowsOf(dto),
		BuffOptions:      buffOptions(),
	}
	panel.Situational, _ = situationalRowsOf(offered, ativos)
	panel.AlwaysOn = alwaysOnRowsOf(flags)
	panel.Count = len(panel.Conditions) + len(panel.Applied) + len(panel.Stances) + activeCountOf(panel.Situational)
	return panel
}

func activeCountOf(linhas []situationalRow) int {
	n := 0
	for _, l := range linhas {
		if l.Active {
			n++
		}
	}
	return n
}

// conditionRowsOf são as condições do livro que estão ligadas.
//
// Id desconhecido é DESCARTADO: o catálogo é a autoridade sobre o que é uma
// condição, e um blob velho não pode injetar uma condição fantasma na ficha.
func conditionRowsOf(dto sheet.CharacterDTO) []conditionRow {
	porID := map[string]book.Condition{}
	for _, c := range book.Catalogs().Condicoes {
		porID[c.ID] = c
	}
	linhas := []conditionRow{}
	for _, id := range sheet.UnmarshalStrings(dto.ActiveConditions) {
		c, conhecida := porID[id]
		if !conhecida {
			continue
		}
		linhas = append(linhas, conditionRow{
			ID: c.ID, Name: c.Name, Effect: c.Description, Page: c.BookPage, Command: c.ID,
		})
	}
	sort.SliceStable(linhas, func(a, b int) bool { return linhas[a].Name < linhas[b].Name })
	return linhas
}

func conditionOptionsFor(dto sheet.CharacterDTO) []pickerOption {
	ligadas := sheet.ToStringSet(sheet.UnmarshalStrings(dto.ActiveConditions))
	opcoes := []pickerOption{}
	for _, c := range book.Catalogs().Condicoes {
		if ligadas[c.ID] {
			continue
		}
		opcoes = append(opcoes, pickerOption{ID: c.ID, Label: c.Name, Detail: c.Description, Command: c.ID})
	}
	return opcoes
}

// stanceRowsOf são as posturas em curso.
func stanceRowsOf(dto sheet.CharacterDTO) []stanceRow {
	doLivro := stancesFromCatalog()
	linhas := []stanceRow{}
	for _, s := range dto.Stances {
		nome := s.Flag
		if posture, conhecida := doLivro[s.Flag]; conhecida {
			nome = posture.Name
		}
		linha := stanceRow{Flag: s.Flag, Name: nome, Command: s.Flag}
		if s.PmPaid > 0 {
			linha.Paid = strconv.FormatInt(s.PmPaid, 10) + " PM"
		}
		if s.Steps > 0 {
			linha.Steps = "+" + strconv.FormatInt(s.Steps, 10)
		}
		linhas = append(linhas, linha)
	}
	return linhas
}

// appliedEffectRowsOf são os consumíveis e as magias de bônus em curso.
func appliedEffectRowsOf(dto sheet.CharacterDTO) []appliedEffectRow {
	linhas := []appliedEffectRow{}
	for _, e := range dto.ActiveEffects {
		linhas = append(linhas, appliedEffectRow{
			ID:        e.ID,
			Name:      effectDisplayName(e.CatalogID),
			Scope:     scopeLabel(e.Scope),
			Modifiers: modifierRowsOf(e.Modifiers),
			Command:   strconv.FormatInt(e.ID, 10),
		})
	}
	return linhas
}

// effectDisplayName troca o id do catálogo pelo nome que a mesa lê.
func effectDisplayName(catalogID string) string {
	for _, m := range book.Catalogs().Magias {
		if m.ID == catalogID {
			return m.Name
		}
	}
	return catalogID
}

// scopeLabel diz até quando o efeito vale.
func scopeLabel(scope string) string {
	switch scope {
	case "day", "dia":
		return "até o fim do dia"
	case "scene", "cena":
		return "até o fim da cena"
	}
	return scope
}

// modifierRowsOf traduz o blob de modificadores em linhas legíveis.
func modifierRowsOf(bruto string) []breakdownRow {
	var mods []engine.Modifier
	if err := json.Unmarshal([]byte(bruto), &mods); err != nil {
		return nil
	}
	linhas := make([]breakdownRow, 0, len(mods))
	for _, m := range mods {
		linhas = append(linhas, breakdownRow{
			Label: targetLabel(m.Target), Value: book.WithSign(m.Amount), Note: m.Note,
		})
	}
	return linhas
}

// targetLabel nomeia o que o modificador toca, em português.
//
// O motor fala `{k: "attack", scope: "all"}`, que é fronteira; quem lê a ficha
// lê "ataque". Alvo sem tradução assentada cai no próprio `k`, que é feio e
// honesto — melhor que inventar um nome que não é o do livro.
//
// A tabela cresceu na fatia 7 (ALE-272) porque a Mochila desenha o que um item
// CONCEDE, e aí aparecem alvos que nenhuma condição usa — `inventorySlots`,
// `spellDC`, `maneuver`, `critRange`. Ela é a mesma lista do
// `describeModifierTarget` do front, e é o único lugar do Go que a tem.
func targetLabel(t engine.ModifierTarget) string {
	nomes := map[string]string{
		"attack": "Ataque", "damage": "Dano", "defense": "Defesa",
		"expertise": "Perícia", "expertiseAll": "Todas as perícias",
		"expertiseRemovePenalty": "Remove penalidade em", "expertiseByAttribute": "Perícias de",
		"attribute": "Atributo", "maxPv": "PV máximo", "maxPm": "PM máximo",
		"displacement": "Deslocamento", "damageReduction": "Redução de dano",
		"defenseDexCap": "Limite de Des na Defesa", "resistance": "Resistências",
		"fearResistance": "Resistência a medo", "critRange": "Margem de ameaça",
		"critMult": "Multiplicador crítico", "pmLimit": "Limite de PM por magia",
		"pmCost": "Custo em PM", "catalyst": "Catalisador", "spellDC": "CD de magias",
		"inventorySlots": "Espaços de carga", "flySpeed": "Voo",
		"armorPenalty": "Penalidade de armadura", "armorPenaltyExpertises": "Penalidade em perícias",
		"tempHp": "PV temporários", "tempMp": "PM temporários", "maneuver": "Manobra",
	}
	// FLAG é booleana e o rótulo dela é uma frase inteira ("Fadiga ao dormir"),
	// não um alvo com complemento — por isso ela sai antes do resto.
	if t.K == "flag" {
		if rotulo, conhecida := itemFlagLabel[t.Name]; conhecida {
			return rotulo
		}
		return t.Name
	}
	nome, tem := nomes[t.K]
	if !tem {
		nome = t.K
	}
	if complemento := targetComplement(t); complemento != "" {
		return nome + " (" + complemento + ")"
	}
	return nome
}

// oComplementoDoAlvo é o que vem entre parênteses depois do alvo.
//
// O escopo `this` NÃO vira texto: o crachá está desenhado no próprio item, e
// "Ataque (deste item)" repete em palavras o que a posição já diz. Os outros
// escopos aparecem traduzidos — deixá-los crus poria `all` e `melee` na tela de
// alguém que joga em português.
func targetComplement(t engine.ModifierTarget) string {
	if t.Name != "" {
		return t.Name
	}
	if t.Attribute != "" {
		return t.Attribute
	}
	if t.School != "" {
		return t.School
	}
	escopos := map[string]string{"all": "todos", "melee": "corpo a corpo", "ranged": "à distância"}
	return escopos[t.Scope]
}

// buffOptions são as magias com efeito aplicável.
func buffOptions() []pickerOption {
	opcoes := []pickerOption{}
	for _, m := range book.Catalogs().Magias {
		spell, conhecida := catalog.LookupSpell(m.ID)
		if !conhecida || spell.Buff == nil {
			continue
		}
		opcoes = append(opcoes, pickerOption{
			ID:      m.ID,
			Label:   m.Name,
			Detail:  circleLabel(m.Circle) + " · " + scopeLabel(spell.Buff.DefaultScope),
			Command: m.ID,
		})
	}
	return opcoes
}

func circleLabel(circle int) string {
	return strconv.Itoa(circle) + "º círculo"
}

// situationalRowsOf agrupa os condicionais que o motor oferece.
//
// # Quem compartilha FLAG vira UM interruptor
//
// Um item caseiro com três modificadores é uma coisa só na mesa; como três
// linhas, a pessoa deixaria metade do efeito ligado. As POSTURAS ficam de fora:
// o interruptor delas mora nos Poderes, porque entrar custa PM.
func situationalRowsOf(offered []engine.ConditionalEffect, ativos map[string]bool) ([]situationalRow, []alwaysOnRow) {
	posturas := stancesFromCatalog()
	porFlag := map[string][]engine.ConditionalEffect{}
	ordem := []string{}
	soltos := []engine.ConditionalEffect{}
	for _, c := range offered {
		if c.Flag == "" {
			soltos = append(soltos, c)
			continue
		}
		if _, ehPostura := posturas[c.Flag]; ehPostura {
			continue
		}
		if _, visto := porFlag[c.Flag]; !visto {
			ordem = append(ordem, c.Flag)
		}
		porFlag[c.Flag] = append(porFlag[c.Flag], c)
	}

	linhas := []situationalRow{}
	for _, c := range soltos {
		id := engine.ConditionalID(c)
		linhas = append(linhas, situationalRow{
			Key: id, Label: conditionalLabel(c), Source: c.Source, Active: ativos[id],
			Modifiers: []breakdownRow{{Label: targetLabel(c.Target), Value: book.WithSign(c.Amount)}},
			Command:   id,
		})
	}
	for _, flag := range ordem {
		grupo := porFlag[flag]
		linha := situationalRow{
			Key: engine.ConditionalID(grupo[0]), Label: conditionalLabel(grupo[0]),
			Source: grupo[0].Source, Folded: len(grupo) > 1,
			Active:  ativos[engine.ConditionalID(grupo[0])],
			Command: engine.ConditionalID(grupo[0]),
		}
		for _, c := range grupo {
			linha.Modifiers = append(linha.Modifiers, breakdownRow{
				Label: targetLabel(c.Target), Value: book.WithSign(c.Amount),
			})
		}
		linhas = append(linhas, linha)
	}
	return linhas, nil
}

// itemFlagLabel é o pt-BR de cada flag sempre ativa.
//
// Ela veio do `ITEM_FLAG_LABEL` do front, onde as MESMAS seis frases já viviam
// em DOIS arquivos — o próprio comentário de lá registra a duplicata e pede a
// consolidação. Aqui elas ficam ao lado de quem as desenha, e um teste do front
// compara as duas cópias enquanto a SPA viver.
//
// Flag desconhecida cai no próprio id, que é feio e HONESTO: inventar uma frase
// para uma flag nova seria pior, porque a tela diria com confiança algo que
// ninguém escreveu.
var itemFlagLabel = map[string]string{
	"lethal-unarmed":              "Ataques desarmados causam dano letal",
	"cannot-apply-dex-to-defense": "Não soma Destreza na Defesa",
	"fatigue-on-sleep":            "Fadiga ao dormir",
	"reach-extends":               "Alcance ampliado",
	"armadura-pesada":             "Conta como armadura pesada",
	"auto-fail-reflexos":          "Falha automática em Reflexos",
}

// alwaysOnRowsOf são as flags de item equipado que NÃO têm interruptor.
//
// A "Fadiga ao dormir" da armadura pesada é o caso: ela está ligada porque a
// armadura está vestida, e não porque alguém a escolheu. Mostrá-las em modo de
// leitura é o que impede este bloco de contradizer o aviso que a MESMA flag
// produz no cabeçalho da ficha — sem elas, a pessoa lê "nenhum efeito" embaixo
// de um aviso que existe por causa de um.
//
// Quem as CALCULA é o motor (`ComputeEquippedFlags`), e não uma varredura
// escrita aqui: ele resolve as condições de uso (vestido, empunhado) e sabe
// quais modificadores do item contam. A primeira versão desta função varria os
// itens à mão e ignorava isso — a tela mostrava o id cru da flag porque a
// tradução também estava faltando, e as duas coisas se escondiam uma na outra.
func alwaysOnRowsOf(flags []engine.EquippedFlag) []alwaysOnRow {
	linhas := []alwaysOnRow{}
	for _, f := range flags {
		rotulo, conhecida := itemFlagLabel[f.Flag]
		if !conhecida {
			rotulo = f.Flag
		}
		linhas = append(linhas, alwaysOnRow{Label: rotulo, Source: f.Source})
	}
	return linhas
}

// conditionalLabel é a frase que descreve QUANDO o modificador vale.
//
// O motor já monta essa nota (`describeCondition`), então aqui não há tradução
// nova — há a queda para o alvo quando a nota vem vazia, que é o caso de um
// modificador caseiro sem texto.
func conditionalLabel(c engine.ConditionalEffect) string {
	if nota := strings.TrimSpace(c.Note); nota != "" {
		return nota
	}
	return targetLabel(c.Target)
}

// osEfeitosAtivosEscrito é a linha que o leitor de tela ouve no lugar da pílula
// de contagem, com o singular certo. Um "1" solto é lido como "1", e o número
// sozinho não diz de quê.
func activeWrittenEffects(n int) string {
	if n == 1 {
		return "1 efeito ativo"
	}
	return strconv.Itoa(n) + " efeitos ativos"
}
