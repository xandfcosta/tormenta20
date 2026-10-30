package sheetui

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
	"t20engine/serve/web/ds"
	"t20engine/serve/web/ui"
)

// A FICHA PARALELA (ALE-473): a ficha vestida no Tempestade, construída aba
// por aba no MESMO endereço com a marca `?tempestade=1`, enquanto a do
// grimório fica intacta. No dia em que a última aba migrar, a marca some e a
// ficha do grimório é apagada — e este arquivo encolhe para o que for casca.

// tempestadeAsked diz se o pedido é da ficha paralela. A marca viaja na query
// pela razão do `?tab=`: o handler descobre o que desenhar lendo a requisição,
// e todo comando redesenha a cena inteira.
func tempestadeAsked(r *http.Request) bool {
	return r.URL.Query().Get("tempestade") == "1"
}

// datastarAsked diz que o pedido veio de um gesto do Datastar (`@get`), e não
// de uma navegação: o Datastar marca os pedidos dele com este cabeçalho.
func datastarAsked(r *http.Request) bool {
	return r.Header.Get("Datastar-Request") == "true"
}

// sheetSystem é o sistema que a página da ficha veste.
func sheetSystem(v View) ui.DesignSystem {
	if v.Tempestade {
		return ui.SystemTempestade
	}
	return ui.SystemGrimoire
}

// tabRoute é o endereço de uma aba, levando a marca quando a ficha é a
// paralela. As abas são LINKS pela razão de sempre (`?tab=` é endereço
// guardado).
func tabRoute(v View, tab string) string {
	route := sheetRoute(v.ID, tab)
	if v.Tempestade {
		route += "&tempestade=1"
	}
	return route
}

// bottomNavPlace é um lugar da barra de baixo.
type bottomNavPlace struct {
	Tab     Tab
	Href    string
	Current bool
	// Count é o número no ícone — só o de Efeitos, que conta o que está em vigor.
	Count int
}

// bottomNavPlaces são os cinco links da barra; o sexto é o "Mais", que é
// botão. O quinto depende de quem CONJURA (decisão do dono): quem conjura tem
// Magias à mão, e quem não conjura tem a Mochila — e o que sai vai para o Mais,
// porque as concedidas por poder também moram em Magias.
func bottomNavPlaces(v View) []bottomNavPlace {
	order := []string{"expertises", "combat", "conditionals", "abilities", fifthPlace(v)}
	places := make([]bottomNavPlace, 0, len(order))
	for _, value := range order {
		tab := tabByValue(v, value)
		place := bottomNavPlace{Tab: tab, Href: tabRoute(v, value), Current: tab.Active}
		if value == "conditionals" {
			place.Count = v.Effects.Count
		}
		places = append(places, place)
	}
	return places
}

// moreTabs são as que moram no "Mais", na ordem das sete.
func moreTabs(v View) []Tab {
	inBar := map[string]bool{"expertises": true, "combat": true, "conditionals": true, "abilities": true, fifthPlace(v): true}
	var more []Tab
	for _, tab := range v.Tabs {
		if !inBar[tab.Value] {
			more = append(more, tab)
		}
	}
	return more
}

// moreIsCurrent marca o "Mais" quando a aba aberta é uma das dele: sem isso,
// com a Mochila aberta pelo Mais, nenhum lugar da barra diria onde a pessoa está.
func moreIsCurrent(v View) bool {
	for _, tab := range moreTabs(v) {
		if tab.Active {
			return true
		}
	}
	return false
}

func fifthPlace(v View) string {
	if v.Spells.IsCaster {
		return "spells"
	}
	return "bag"
}

func tabByValue(v View, value string) Tab {
	for _, tab := range v.Tabs {
		if tab.Value == value {
			return tab
		}
	}
	panic(fmt.Sprintf("aba %q não existe; as sete estão em Tabs()", value))
}

// activeEffect é um efeito em vigor com o gesto que o tira e a descrição que
// o nome abre, para a coluna da ficha: condição, postura e efeito aplicado. O
// situacional fica de fora — ele é interruptor da aba Efeitos, e não algo que
// "está em você".
type activeEffect struct {
	// Kind é "condition", "stance" ou "applied": a aba Efeitos separa a
	// condição do que está em vigor.
	Kind   string
	Name   string
	Remove string
	// Dialog é o `id` da descrição; Meta é a linha de cima dela
	// ("Condição · p394"), Text o que o efeito faz, e Terms os modificadores
	// do efeito aplicado.
	Dialog string
	Meta   string
	Text   string
	Terms  []ds.SumTerm
}

func activeEffects(v View) []activeEffect {
	var out []activeEffect
	for _, c := range v.Effects.Conditions {
		out = append(out, activeEffect{
			Kind: "condition", Name: c.Name, Remove: sheetPost(v, "/efeitos/condicao/"+c.Command),
			Dialog: "effect-condition-" + c.ID, Meta: withPage("Condição", c.Page), Text: c.Effect,
		})
	}
	for _, s := range v.Effects.Stances {
		effect := activeEffect{
			Kind: "stance", Name: s.Name, Remove: sheetPost(v, "/efeitos/postura/"+s.Command),
			Dialog: "effect-stance-" + s.Flag, Meta: stanceMeta(s),
		}
		// O TEXTO DA POSTURA é o do poder que a concede: a postura em curso só
		// sabe o que foi pago e o que ela dá.
		if power, ok := stancePower(v, s.Flag); ok {
			effect.Meta = withPage(effect.Meta, power.Page)
			effect.Text = power.Detail
		}
		out = append(out, effect)
	}
	for _, a := range v.Effects.Applied {
		out = append(out, activeEffect{
			Kind: "applied", Name: a.Name, Remove: sheetPost(v, "/efeitos/encerra/"+a.Command),
			Dialog: "effect-applied-" + strconv.FormatInt(a.ID, 10), Meta: strings.TrimSuffix("Efeito · "+a.Scope, " · "),
			Text: a.Detail, Terms: sumTermsOf(a.Modifiers),
		})
	}
	return out
}

// stanceMeta é "Postura · pago 2 PM · +3".
func stanceMeta(s stanceRow) string {
	parts := []string{"Postura"}
	if s.Paid != "" {
		parts = append(parts, "pago "+s.Paid)
	}
	if s.Bonus != "" {
		parts = append(parts, s.Bonus)
	}
	return strings.Join(parts, " · ")
}

// stancePower acha o poder que concede a postura, pela flag dela.
func stancePower(v View, flag string) (powerRow, bool) {
	for _, row := range v.Powers.Actions {
		if row.Stance != nil && row.Stance.Flag == flag {
			return row, true
		}
	}
	return powerRow{}, false
}

// withPage acrescenta a página à linha de cima, quando há uma.
func withPage(meta string, page int) string {
	if page <= 0 {
		return meta
	}
	return meta + " · p" + strconv.Itoa(page)
}

// vitalAmountPost é o gesto do diálogo de dano ou cura: o VALOR digitado entra
// no caminho, com o sinal do botão. A rota de passo já aceita qualquer inteiro
// diferente de zero e prende na faixa, então "levou 12" é o mesmo gesto do
// −1, com outro número. O zero não sai: um passo zero é recusado pela rota.
func vitalAmountPost(v View, which string, sign int) string {
	value := "$vital_amount"
	if sign < 0 {
		value = "-" + value
	}
	return "$vital_amount > 0 && " + sheetPostWith(v, "/vitais/"+which+"/§", value)
}

// whoLine é a linha de quem é, debaixo do nome: classe, raça e devoção.
func whoLine(v View) string {
	parts := []string{}
	if v.Classes != "" {
		parts = append(parts, v.Classes)
	}
	if v.Summary != "" {
		parts = append(parts, v.Summary)
	}
	return strings.Join(parts, " · ")
}

// identityLine é uma linha de quem é, no ⋯.
type identityLine struct {
	Label string
	Value string
}

// identityOf é quem o personagem é, por extenso: no celular o cabeçalho só
// cabe o nome, e isto mora no ⋯ (decisão do dono). Personagem sem classe,
// raça ou origem ainda está sendo feito, e a linha diz o que falta em vez de
// sumir.
func identityOf(dto sheet.CharacterDTO, classes string) []identityLine {
	races := make([]string, 0, len(dto.Races))
	for _, r := range dto.Races {
		races = append(races, book.RaceLabel(r.Race))
	}
	god := "Nenhuma"
	if dto.God != nil && *dto.God != "" {
		god = *dto.God
	}
	return []identityLine{
		{"Classe", orMissing(classes)},
		{"Raça", orMissing(strings.Join(races, " / "))},
		{"Divindade", god},
		{"Origem", orMissing(book.OriginLabel(dto.Origin))},
		{"Tamanho", orMissing(dto.Size)},
	}
}

func orMissing(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}

// meterOf traduz o vital da ficha para o medidor do kit. O temporário chega
// escrito ("+6"), porque é assim que a ficha do grimório o mostra.
func meterOf(label string, vital sheetVital, kind ds.MeterKind) ds.Meter {
	temp, _ := strconv.Atoi(strings.TrimPrefix(vital.Temp, "+"))
	return ds.Meter{Label: label, Value: int(vital.Current), Max: int(vital.Max), Temp: temp, Kind: kind}
}
