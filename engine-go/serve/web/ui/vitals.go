package ui

import (
	"strconv"

	"t20engine/domain/engine"
)

// A ESCADA DO PV: a COR diz "quão mal", e não só a largura.
//
// Ela mora no kit porque é lida por QUATRO superfícies que não se importam entre
// si — a Mesa, o tabuleiro, a ficha e a lista de heróis — e porque é regra de
// APRESENTAÇÃO: quanto de vida vira qual tinta da casa. O `web/ui` é o único
// pacote que as quatro já importam.
//
// Privada de um pacote, uma regra não diverge — ela simplesmente NÃO ALCANÇA, e
// quem precisa dela escreve outra coisa: enquanto a escada foi privada do
// `web/table`, a ficha e a lista de heróis escreviam `--hp-full` fixo, e o mesmo
// herói a 17,5% de PV saía verde numa tela e vermelho na outra.

// hpCritical e hpHurt são os limiares, em porcento.
//
// Escritos como constantes porque são a REGRA e não números soltos: quem mudar
// um deles está mudando o que a mesa chama de "ferido", e isso se faz em um
// lugar só.
const (
	hpCritical = 25
	hpHurt     = 50
)

// VitalPercent é a largura da barra, presa em 0..100.
//
// Máximo ausente ou zero devolve ZERO em vez de dividir: quem não tem mana não
// tem barra cheia nem vazia — ele não tem barra, e quem chama decide não
// desenhar.
//
// Exemplo:
//
//	ui.VitalPercent(10, 57) // → 17
func VitalPercent(current, max int64) int {
	if max <= 0 {
		return 0
	}
	pct := int(current * 100 / max)
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

// HpFillTone é a classe que PREENCHE a barra de PV.
//
// Exemplo:
//
//	class={ "h-full", ui.HpFillTone(v.Porcento) } // → "h-full bg-hp-critical"
func HpFillTone(pct int) string {
	switch {
	case pct <= hpCritical:
		return "bg-hp-critical"
	case pct <= hpHurt:
		return "bg-hp-hurt"
	}
	return "bg-hp-full"
}

// HpInkTone é a classe que ESCREVE o PV, e ela NÃO é a de preencher.
//
// Os três tons foram escolhidos como cor de BARRA, e dois deles servem de texto
// por acaso: sobre o painel o verde dá 5,34:1 e o âmbar 6,25:1, mas o
// `--hp-critical` dá 4,11:1 — abaixo do mínimo de texto pequeno. Exatamente na
// hora em que a barra grita "este aqui está morrendo", o número ao lado dela
// seria o menos legível da tela.
//
// O crítico escreve com a tinta de perigo da casa, que dá 5,21:1 sobre o painel.
// Tinta própria seria um segundo vermelho quase idêntico ao lado do primeiro, e
// a casa tem uma palavra por conceito. É a mesma divisão que o marcador do
// tabuleiro faz, pela mesma razão e com o mesmo número.
//
// Exemplo:
//
//	class={ "font-mono", ui.HpInkTone(pct) } // → "font-mono text-grimorio-crimson-bright"
func HpInkTone(pct int) string {
	switch HpFillTone(pct) {
	case "bg-hp-critical":
		return "text-grimorio-crimson-bright"
	case "bg-hp-hurt":
		return "text-hp-hurt"
	}
	return "text-hp-full"
}

// ── abaixo de 1 PV (T20 p236, ALE-366) ─────────────────────────────────────

// DownedWord é a palavra do estado de quem está a 0 PV ou menos — "morrendo",
// "estável" ou "morto" —, e vazio para quem está de pé.
//
// QUAL estado é decisão do motor (`engine.VitalStateOf`); aqui só se escolhe a
// palavra, como a escada escolhe a cor. Ela é só para FICHA: o livro não diz
// que o capanga do mestre sangra do mesmo jeito, e a fila não guarda condição
// dele — uma palavra ali seria inventada.
//
// Exemplo:
//
//	ui.DownedWord(-43, 87, false) // → "morto"
func DownedWord(hp, hpMax int64, bleeding bool) string {
	switch engine.VitalStateOf(hp, hpMax, bleeding) {
	case engine.VitalDying:
		return "morrendo"
	case engine.VitalStable:
		return "estável"
	case engine.VitalDead:
		return "morto"
	}
	return ""
}

// DownedInk é a tinta da palavra, e ela separa o que pede AÇÃO do que é fato.
// Morrendo é o único que o mestre ainda muda — cura, teste de Cura, o d20 da vez
// —, e leva a tinta de perigo da casa (a do número crítico, pelo contraste que o
// `HpInkTone` explica). Morto é final e sai na tinta de texto: com o mesmo
// vermelho, a fila gritava "urgente" sobre quem não tem mais o que fazer.
// Estável é informação e sai apagado.
func DownedInk(word string) string {
	switch word {
	case "morrendo":
		return "text-grimorio-crimson-bright"
	case "morto":
		return "text-foreground"
	}
	return "text-muted-foreground"
}

// HitPoints escreve o PV com o SINAL DE MENOS tipográfico (U+2212), o mesmo
// dos botões de passo ao lado: com o hífen, "−5" e "-43" ficavam lado a lado
// com dois traços diferentes para a mesma ideia.
//
// Exemplo:
//
//	ui.HitPoints(-43) // → "−43"
func HitPoints(n int64) string {
	if n < 0 {
		return "−" + strconv.FormatInt(-n, 10)
	}
	return strconv.FormatInt(n, 10)
}
