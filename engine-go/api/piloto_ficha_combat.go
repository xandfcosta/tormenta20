package api

import (
	"strconv"

	"t20engine/engine"
	"t20engine/sheet"
)

// A aba COMBATE como dado (ALE-272, fatia 3).
//
// São quatro blocos, na ordem em que a SPA os desenha: os três números que se
// olha no meio do turno (Defesa e os dois ataques), as três resistências, os
// seis atributos, e o que só às vezes importa — as fórmulas de arma de quem
// empunha alguma e a tripla de quem conjura.
//
// # O painel é de LEITURA, e isso muda o desenho inteiro
//
// Nenhum botão daqui escreve no banco: não há `@post`, e por isso a sexta
// armadilha do Datastar (o comando que perde o `?tab=`) não tem onde morder
// nesta fatia. A aba inteira é UM `GET`, e os diálogos de decomposição abrem
// pelo cliente sem pedir nada ao servidor — ver `oPainelDeCombate`.
//
// # Esta é a primeira conta do servidor com os condicionais LIGADOS
//
// Todo uso anterior do motor no piloto passou pelo `sheetFromDTO`, que computa a
// ficha BASE (`map[string]bool{}`). Aqui isso mentiria: um bárbaro em Fúria veria
// o ataque de quem não está em Fúria, e a ficha discordaria da Mesa, que já lê o
// estado ligado. O opt-in do jogador vem do banco dentro do próprio DTO
// (`loadPlayState`), então não há segunda fonte a consultar.
//
// É também a primeira chamada de `ComputeWeaponCards` fora do WASM: até agora só
// o navegador montava cartão de arma.

// combatPanel é a aba Combate pronta para desenhar.
type combatPanel struct {
	// Tiles são Defesa, Atq CaC e Atq Dist; Saves são as três resistências. Duas
	// listas e não uma porque elas são DUAS fileiras de três na tela, e juntá-las
	// deixaria a quebra de linha por conta da largura.
	Tiles []statTile
	Saves []statTile
	// Attributes não abrem diálogo — são leitura seca, como na SPA.
	Attributes []attributeTile
	// ShowWeapons segue a regra da SPA, que não é óbvia: quem empunha vê os
	// cartões; o marcial de mãos livres vê o texto de vazio, para a caixa não
	// parecer quebrada; e o conjurador puro de mãos livres não vê o bloco, porque
	// para ele a tripla mágica é que é o assunto.
	ShowWeapons bool
	Weapons     []weaponTile
	// MagicTiles é vazia para quem não conjura por CLASSE. Um poder que concede
	// uma magia solta não abre este bloco, e é o mesmo critério da SPA.
	MagicTiles []statTile
}

// statTile é uma caixa com um número e a decomposição dele por trás.
type statTile struct {
	// Key é o que o diálogo compara com o sinal `$detalhe`, e por isso ela é
	// única na página. Ver `oPainelDeCombate`.
	Key   string
	Label string
	// Title é o nome por extenso, para o diálogo: a caixa diz "Atq CaC" porque
	// não cabe mais, e um diálogo com o rótulo abreviado deixaria a abreviação
	// sem nunca ser explicada.
	Title string
	Icon  string
	Value string
	// Sub é a linha pequena sob o número — hoje só "RD 4".
	Sub string
	// Magic troca a paleta para a arcana. A SPA chama isso de `tone`; aqui é um
	// booleano porque são dois tons e não uma família aberta.
	Magic bool
	Rows  []breakdownRow
	// Extra são valores que se RELACIONAM com o número sem somar nele — as
	// fontes da redução de dano, sob a Defesa.
	Extra *extraBlock
}

// breakdownRow é uma linha do diálogo: de onde o número veio, e quanto.
type breakdownRow struct {
	Label string
	Value string
	// Muted é a linha que vale ZERO e mesmo assim aparece, porque a ausência dela
	// é que seria a pergunta: "por que minha Defesa está baixa?" se responde
	// vendo a Destreza bloqueada, não vendo a Destreza sumir.
	Muted bool
	// Note é o PORQUÊ do modificador ("desbalanceada: -2 em ataque"), e ela
	// quebra linha em vez de truncar.
	Note string
	// Indented é a linha que EXPLICA a de cima em vez de somar ao lado dela — as
	// contribuições de item sob o "Outros" das Perícias. O Combate não a usa: lá
	// a lista é plana, porque cada linha é uma parcela do total.
	Indented bool
}

type extraBlock struct {
	Title string
	Rows  []breakdownRow
}

// attributeTile é um dos seis quadrados, sem diálogo.
type attributeTile struct {
	Abbr  string
	Value string
}

// weaponTile é uma arma empunhada como fórmula pronta de rolar.
type weaponTile struct {
	Key    string
	Name   string
	Skill  string
	Attack string
	Damage string
	Crit   string
	// AttackRows e DamageRows são duas seções ROTULADAS no diálogo, e não uma
	// lista com um total: uma arma não tem "total", tem duas contas.
	AttackRows []breakdownRow
	DamageRows []breakdownRow
}

// attributeAbbr são as abreviações que a mesa fala, na ordem do livro.
//
// Elas nascem aqui porque o motor não as tem: ele fala `strength`, que é
// fronteira, e FOR é texto de tela.
var attributeAbbr = map[string]string{
	"strength":     "FOR",
	"dexterity":    "DES",
	"constitution": "CON",
	"intelligence": "INT",
	"wisdom":       "SAB",
	"charisma":     "CAR",
}

// theSaves são as três resistências na ordem em que a mesa as pede.
var theSaves = []struct {
	Name      string
	Attribute string
	Abbr      string
}{
	{"Fortitude", "constitution", "CON"},
	{"Reflexos", "dexterity", "DES"},
	{"Vontade", "wisdom", "SAB"},
}

// combatPanelOf computa a aba Combate de um personagem.
//
// SEM CATÁLOGO PRIMADO ela devolve o painel vazio, pela mesma razão que a Defesa
// do crachá vira travessão: a ficha inteira não pode deixar de abrir por causa
// de um número, e a aba diz que não sabe em vez de mostrar zeros — um zero é um
// valor plausível, e o jogador agiria sobre ele.
// sheetForPanels computa a ficha UMA vez para os painéis que a leem.
//
// Combate e Perícias precisam do MESMO resultado, e computar duas vezes daria
// duas contas que podem divergir no dia em que uma delas passar um conjunto de
// condicionais diferente do da outra — o jogador veria a Luta com um número no
// Combate e outro nas Perícias.
//
// O segundo retorno diz se houve conta: sem catálogo primado não há ficha, e o
// painel que chamar desenha o que sabe desenhar sem ela.
func (s *Server) sheetForPanels(dto sheet.CharacterDTO) (engine.ComputedSheetV2, []engine.WeaponCard, bool) {
	if s.catalogs == nil {
		return engine.ComputedSheetV2{}, nil, false
	}
	ec, err := engineCharacterFrom(dto)
	if err != nil {
		return engine.ComputedSheetV2{}, nil, false
	}
	// O OPT-IN DO JOGADOR entra na conta, e é a primeira vez que uma cena do
	// piloto o faz: todo uso anterior passou pelo `sheetFromDTO`, que computa a
	// ficha base. Com Fúria ligada, a base mostraria o ataque de quem não está
	// em Fúria e a ficha discordaria da Mesa.
	active := toStringSet(dto.Conditionals)
	return s.catalogs.ComputeSheetV2(ec, active), s.catalogs.ComputeWeaponCards(ec, active), true
}

func (s *Server) combatPanelOf(dto sheet.CharacterDTO) combatPanel {
	sheet, cards, ok := s.sheetForPanels(dto)
	if !ok {
		return combatPanel{}
	}
	return combatPanelFor(sheet, cards, isCaster(sheet))
}

// combatPanelFor monta a aba inteira.
//
// Ela recebe a ficha JÁ computada em vez de computar: o `carregaFicha` chama o
// motor uma vez e reparte, e computar de novo aqui daria duas contas que podem
// divergir no dia em que uma delas passar condicional diferente.
func combatPanelFor(sheet engine.ComputedSheetV2, cards []engine.WeaponCard, caster bool) combatPanel {
	return combatPanel{
		Tiles:       defenseAndAttackTiles(sheet),
		Saves:       saveTiles(sheet),
		Attributes:  attributeTiles(sheet),
		ShowWeapons: len(cards) > 0 || !caster,
		Weapons:     weaponTiles(cards),
		MagicTiles:  magicTiles(sheet, caster),
	}
}

// defenseAndAttackTiles são os três números do meio do turno.
func defenseAndAttackTiles(sheet engine.ComputedSheetV2) []statTile {
	luta := expertiseOrZero(sheet, "Luta", "strength")
	pontaria := expertiseOrZero(sheet, "Pontaria", "dexterity")
	return []statTile{
		defenseTile(sheet),
		attackTile("attack-melee", "Atq CaC", "Ataque Corpo a Corpo (Luta)", luta, sheet.AttackAll),
		attackTile("attack-ranged", "Atq Dist", "Ataque à Distância (Pontaria)", pontaria, sheet.AttackAll),
	}
}

// defenseTile é a Defesa, com a redução de dano pendurada quando existe.
func defenseTile(sheet engine.ComputedSheetV2) statTile {
	tile := statTile{
		Key: "defense", Label: "Defesa", Title: "Defesa", Icon: "Shield",
		Value: strconv.Itoa(sheet.Defense.Total),
		Rows:  defenseRows(sheet),
	}
	rd := sheet.DamageReduction
	if rd.Total > 0 {
		tile.Sub = "RD " + strconv.Itoa(rd.Total)
		tile.Extra = &extraBlock{
			Title: "Redução de dano " + strconv.Itoa(rd.Total),
			Rows:  rowsFromSourceAmounts(rd.Sources),
		}
	}
	return tile
}

// attackTile é um dos dois ataques, já somado ao que vale para TODOS eles.
func attackTile(key, label, title string, ex engine.ExpertiseBreakdown, all engine.TotalContribs) statTile {
	return statTile{
		Key: key, Label: label, Title: title, Icon: iconForAttack(key),
		Value: comSinalInt(ex.Total + all.Total),
		Rows:  expertiseRows(ex, &all),
	}
}

func iconForAttack(key string) string {
	if key == "attack-ranged" {
		return "Crosshair"
	}
	return "Sword"
}

// saveTiles são Fortitude, Reflexos e Vontade.
//
// O rótulo da caixa é cortado em quatro letras como na SPA ("Fort", "Refl",
// "Vont") — é o que cabe em três colunas num telefone —, e o nome inteiro vai no
// diálogo.
func saveTiles(sheet engine.ComputedSheetV2) []statTile {
	tiles := make([]statTile, 0, len(theSaves))
	for _, save := range theSaves {
		ex := expertiseOrZero(sheet, save.Name, save.Attribute)
		tiles = append(tiles, statTile{
			Key: "save-" + save.Abbr, Label: shortSaveLabel(save.Name), Title: save.Name,
			Icon: "ShieldCheck", Value: comSinalInt(ex.Total), Rows: expertiseRows(ex, nil),
		})
	}
	return tiles
}

// shortSaveLabel corta o nome no que cabe na caixa.
func shortSaveLabel(name string) string {
	if len(name) <= 4 {
		return name
	}
	return string([]rune(name)[:4])
}

// attributeTiles são os seis, na ordem do livro.
func attributeTiles(sheet engine.ComputedSheetV2) []attributeTile {
	tiles := make([]attributeTile, 0, len(engine.AttributeKeys))
	for _, key := range engine.AttributeKeys {
		tiles = append(tiles, attributeTile{
			Abbr:  attributeAbbr[key],
			Value: comSinalInt(sheet.Attributes[key].Total),
		})
	}
	return tiles
}

// magicTiles é a tripla do conjurador, ou nada.
func magicTiles(sheet engine.ComputedSheetV2, caster bool) []statTile {
	if !caster {
		return nil
	}
	return []statTile{
		{Key: "pm-limit", Label: "Limite PM", Title: "Limite de PM por magia", Icon: "Zap",
			Magic: true, Value: strconv.Itoa(sheet.PmLimit.Total), Rows: pmLimitRows(sheet)},
		{Key: "spell-dc", Label: "CD Magia", Title: "CD dos testes de resistência das suas magias",
			Icon: "Sparkles", Magic: true, Value: strconv.Itoa(spellDcTotal(sheet)), Rows: spellDcRows(sheet)},
		{Key: "pm-cost", Label: "Custo PM", Title: "Modificador de custo de PM", Icon: "Sparkles",
			Magic: true, Value: comSinalInt(sheet.PmCostMod.Total), Rows: pmCostRows(sheet)},
	}
}

// spellDcTotal é a CD que a mesa anuncia: a base do motor mais o que os itens
// somam.
func spellDcTotal(sheet engine.ComputedSheetV2) int {
	base := 0
	if sheet.BestBaseSpellCd != nil {
		base = *sheet.BestBaseSpellCd
	}
	return base + sheet.SpellDCBonus.Total
}

// weaponTiles são as armas empunhadas, no máximo duas — o motor já as limita.
func weaponTiles(cards []engine.WeaponCard) []weaponTile {
	tiles := make([]weaponTile, 0, len(cards))
	for i, card := range cards {
		attackRows, damageRows := weaponRows(card)
		tiles = append(tiles, weaponTile{
			Key:        "weapon-" + strconv.Itoa(i),
			Name:       card.Name,
			Skill:      card.Skill,
			Attack:     comSinalInt(card.Attack),
			Damage:     damageLabel(card),
			Crit:       critLabel(card),
			AttackRows: attackRows,
			DamageRows: damageRows,
		})
	}
	return tiles
}

// damageLabel é "1d8+4": o dado da arma mais o que se soma a ele, e só o dado
// quando não há soma nenhuma.
func damageLabel(card engine.WeaponCard) string {
	if card.DamageBonus == 0 {
		return card.Damage
	}
	return card.Damage + comSinalInt(card.DamageBonus)
}

// critLabel é "19-20/x3". Margem 20 se escreve sozinha, e não como "20-20".
func critLabel(card engine.WeaponCard) string {
	margin := "20"
	if card.CritRange < 20 {
		margin = strconv.Itoa(card.CritRange) + "-20"
	}
	return margin + "/x" + strconv.Itoa(card.CritMult)
}

// isCaster diz se o personagem conjura por CLASSE.
//
// A pergunta é do motor e não de uma lista repetida aqui: `BestBaseSpellCd` é
// nulo exatamente para quem não tem classe conjuradora, e ele já resolve o
// Caminho do Arcanista, que uma lista de nomes não resolveria.
func isCaster(sheet engine.ComputedSheetV2) bool {
	return sheet.BestBaseSpellCd != nil
}

// expertiseOrZero acha a perícia na ficha, ou devolve uma zerada.
//
// Zero e não erro: uma ficha sem a linha de Luta desenha "+0" e seis linhas de
// decomposição vazias, que é o que a SPA faz. Derrubar a aba inteira porque uma
// perícia não foi gravada trocaria um número errado por nenhuma tela.
func expertiseOrZero(sheet engine.ComputedSheetV2, name, attribute string) engine.ExpertiseBreakdown {
	for _, ex := range sheet.Expertises {
		if ex.Name == name {
			return ex
		}
	}
	return engine.ExpertiseBreakdown{Name: name, Attribute: attribute}
}
