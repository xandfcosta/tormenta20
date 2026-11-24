package sheetui

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

// O DADO DA VEZ NA LINHA EM VIGOR (ver GLOSSARY, **dado da vez**): a magia em vigor que rola de novo na vez de quem
// conjurou ganha, ao lado do Encerrar, um botão por versão. Ele abre o diálogo com o dado — já com os aprimoramentos
// da conjuração, que o servidor guardou — e o campo do total rolado na mão, pelos mesmos sinais da conjuração.
// Quem diz se é a hora é a mesa: a linha só diz quando vale.

// turnRollOffer é um botão do dado da vez: uma versão da magia, o diálogo dela e as parcelas escritas.
type turnRollOffer struct {
	EffectID int64
	Version  string
	// Button é o texto do botão ("Rolar", ou o nome da versão no Manto), e Label o nome acessível ("Rolar Ferver
	// Sangue", "Rolar Manto de Luz").
	Button string
	Label  string
	Dialog string
	// Title é a cabeça do diálogo: "Ferver Sangue · início do turno".
	Title string
	Lines []turnRollLine
}

// turnRollLine é uma parcela escrita ("5d8 de fogo"), a condição do texto e o campo do total da mão dela.
type turnRollLine struct {
	Written string
	Note    string
	Signal  string
	Label   string
}

// turnRollOffersOf são os botões do dado da vez de uma linha em vigor, vazios na que não é magia com dado da vez.
func turnRollOffersOf(dto sheet.CharacterDTO, effect sheet.EffectDTO) []turnRollOffer {
	onSheet := sheet.CasterOf(dto).Spell(effect.CatalogID)
	if !onSheet.Known || onSheet.Spell.TurnRoll == nil {
		return nil
	}
	at := book.TurnRollAt(onSheet.Spell.TurnRoll.At)
	versions := onSheet.TurnVersions()
	offers := make([]turnRollOffer, 0, len(versions))
	for i, version := range versions {
		parts, err := onSheet.TurnRoll(effect.CastAugments, version)
		if err != nil {
			continue
		}
		name, button := onSheet.Spell.Name, "Rolar"
		if version != "" {
			name, button = version, version
		}
		offer := turnRollOffer{
			EffectID: effect.ID, Version: version, Button: button, Label: "Rolar " + name,
			Dialog: fmt.Sprintf("turn-roll-%d-%d", effect.ID, i), Title: name + " · " + at,
		}
		for j, part := range parts {
			written := part.Dice
			if part.Kind != "" {
				written += " de " + part.Kind
			}
			offer.Lines = append(offer.Lines, turnRollLine{
				Written: written, Note: part.Note, Signal: "spell_hand_" + strconv.Itoa(j+1),
				Label: fmt.Sprintf("O total da parcela %d que você rolou na mesa para %s", j+1, name),
			})
		}
		offers = append(offers, offer)
	}
	return offers
}

// turnRollMeta é o que a linha em vigor diz do dado da vez: "rola no início do turno". Vazia sem dado da vez.
func turnRollMeta(dto sheet.CharacterDTO, effect sheet.EffectDTO) string {
	onSheet := sheet.CasterOf(dto).Spell(effect.CatalogID)
	if !onSheet.Known || onSheet.Spell.TurnRoll == nil {
		return ""
	}
	return "rola no " + book.TurnRollAt(onSheet.Spell.TurnRoll.At)
}

// turnRollCommand é o `@post` do botão Rolar do diálogo. A versão vai depois da aba, porque o `commandRoute` já
// abre a query.
func turnRollCommand(v View, offer turnRollOffer) string {
	route := commandRoute(v, "/efeitos/vez/"+strconv.FormatInt(offer.EffectID, 10))
	if offer.Version != "" {
		route += "&versao=" + url.QueryEscape(offer.Version)
	}
	return fmt.Sprintf("@post('%s')", strings.ReplaceAll(route, "'", "\\'"))
}
