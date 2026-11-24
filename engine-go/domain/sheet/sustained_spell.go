package sheet

import (
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// A MAGIA SUSTENTADA visto pelo conjurador (p227; ver GLOSSARY, **magia sustentada**): se a conjuração é
// sustentada, e a recusa da segunda. O Conjurar, o Aplicar da aba Efeitos e o cadeado das abas fazem a mesma
// pergunta, e a resposta mora aqui para não divergirem.

// SustainedClash é a frase que recusa uma SEGUNDA magia sustentada, ou vazia: "você pode manter diversas
// habilidades sustentadas, pagando o custo de cada uma, mas apenas uma magia sustentada por vez" (p227).
//
// Ela conta a sustentada que ESTA ficha mantém (`EffectDTO.Sustaining`): a magia que outro conjurador lançou e
// só foi aplicada aqui é dele. A recusa NOMEIA a que está de pé, para quem joga saber o que encerrar; reconjurar
// a mesma não é uma segunda.
//
//	sheet.SustainedClash(dto.ActiveEffects, "silencio") // "Tempestade Divina já está sustentada, e o livro permite uma magia sustentada por vez (p227)"
func SustainedClash(effects []EffectDTO, spellID string) string {
	for _, e := range effects {
		if e.CatalogID != spellID && e.Sustaining() {
			return fmt.Sprintf("%s já está sustentada, e o livro permite uma magia sustentada por vez (p227)",
				book.SpellName(e.CatalogID))
		}
	}
	return ""
}

// Sustains diz se a conjuração com estes aprimoramentos é sustentada: a duração do verbete, a menos que um
// aprimoramento escolhido a ponha ou a tire (`book.SpellAugment.Sustains`). Mudanças na mesma característica
// não se acumulam (p171), então no máximo um deles fala.
//
// Índice fora de faixa é ignorado de propósito: quem recusa o pedido malformado é o `AugmentPrice`.
func (s SpellOnTheSheet) Sustains(picks []AugmentPick) bool {
	is := sustained(s.Spell.Duration)
	for _, p := range picks {
		if p.AugmentIndex < 0 || p.AugmentIndex >= len(s.Spell.Augments) {
			continue
		}
		if says := s.Spell.Augments[p.AugmentIndex].Sustains; says != nil {
			is = *says
		}
	}
	return is
}

// CastRefusal é a recusa de conjurar a magia com estes aprimoramentos que a ficha responde sozinha, ou vazia:
// a magia acima do círculo, a do mago não memorizada (p37) e a segunda sustentada (p227). O `CastLock` é ela
// sem aprimoramento, que é o Conjurar das abas. O que depende do momento — a forma selvagem, o PM, o turno — é
// de quem conjura.
//
//	sheet.CasterOf(dto).Spell("invisibilidade").CastRefusal(esfera) // "Velocidade já está sustentada, …"
func (s SpellOnTheSheet) CastRefusal(picks []AugmentPick) string {
	if s.fixedLock != "" {
		return s.fixedLock
	}
	if s.Sustains(picks) {
		return s.SustainedClash
	}
	return ""
}

// LockedBySustained diz se quem tranca o Conjurar sem aprimoramento (`CastLock`) é a outra sustentada de pé,
// e não o círculo nem a memorização: a linha do grimório a marca.
func (s SpellOnTheSheet) LockedBySustained() bool {
	return s.fixedLock == "" && s.Sustains(nil) && s.SustainedClash != ""
}

// Sustaining diz se ESTA ficha sustenta a linha: ela é sustentada e, sendo de magia, o Conjurar desta ficha a
// abriu ou a tocou. É a linha que a manutenção cobra e que conta no limite de uma (p227); a magia só aplicada
// aqui veio de outro conjurador, que é quem paga (decisão do dono; ver GLOSSARY, **magia sustentada**). A
// sustentada que não é magia é de quem a ativou.
//
//	sheet.EffectDTO{Source: "spell", Scope: "sustained"}.Sustaining() // false: foi só aplicada
func (e EffectDTO) Sustaining() bool {
	return sustained(e.Scope) && (e.Source != "spell" || e.CastHere)
}

// sustained diz se a duração escrita é sustentada, na grafia do catálogo ou na da coluna `scope`.
func sustained(written string) bool {
	duration, err := engine.ParseDuration(written)
	return err == nil && duration.Kind == engine.DurationSustained
}
