package sheetui

import (
	"fmt"
	"net/http"
	"strconv"
	"t20engine/domain/sheet"

	"github.com/go-chi/chi/v5"

	"t20engine/domain/engine"
	"t20engine/infra/db/sqlcgen"
)

// OS COMANDOS DA ABA EFEITOS.

// toggleBookCondition liga ou desliga UMA condição do livro (p394-395).
//
// # Ela avisa a MESA, e isso não é enfeite
//
// O motor deriva Defesa e perícias da condição, então uma condição aplicada sem
// aviso faz o jogador e o mestre verem números DIFERENTES do mesmo personagem,
// sem nada na tela dizendo que discordam.
//
// O aviso sai DEPOIS da escrita, nunca antes: avisar sobre algo que ainda pode
// falhar faria a mesa buscar o estado velho e acreditar nele.
func toggleBookCondition(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	if err := s.plays.ToggleBookCondition(r.Context(), row, chi.URLParam(r, "cond")); err != nil {
		return err
	}
	// O AVISO sai DEPOIS da escrita, nunca antes: avisar sobre algo que ainda
	// pode falhar faria a mesa buscar o estado velho e acreditar nele. Ele fica
	// na cena porque depende do barramento do PROCESSO, que é do hospedeiro.
	s.deps.CharacterChanged(row.ID)
	return nil
}

// applySpellBuff aplica uma magia de bônus como efeito de cena ou dia.
//
// A gravação é a do caso de uso (`character.Plays.ApplySpellBuff`): duas
// escritas divergiriam no dia em que uma regra nova chegasse, e o escopo padrão
// de cada magia vive no catálogo, não aqui.
func applySpellBuff(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	_, err := s.plays.ApplySpellBuff(r.Context(), row.ID, chi.URLParam(r, "magia"), nil)
	return err
}

// endAppliedEffect encerra um efeito em curso.
func endAppliedEffect(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	id, err := strconv.ParseInt(chi.URLParam(r, "efeito"), 10, 64)
	if err != nil {
		return fmt.Errorf("o efeito %q não é um número", chi.URLParam(r, "efeito"))
	}
	return s.plays.EndAppliedEffect(r.Context(), row.ID, id)
}

// endStance encerra uma postura.
//
// Encerrar apaga a linha da postura E desliga a flag: são duas escritas para uma
// coisa só, e deixar a flag ligada manteria os modificadores em pé numa postura
// que a tela diz encerrada. Entrar continua sendo dos Poderes, onde o PM é
// cobrado — aqui não há como pagar nada.
func endStance(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	flag := chi.URLParam(r, "flag")
	if err := s.deps.Queries().RemoveCharacterStance(r.Context(), sqlcgen.RemoveCharacterStanceParams{
		Characterid: row.ID, Flag: flag,
	}); err != nil {
		return err
	}
	// E O QUE A POSTURA CONCEDEU sai junto: a reserva de PV temporários da Alma
	// de Bronze dura "enquanto a Fúria durar" (p41), e deixá-la para trás daria
	// PV que a postura encerrada continua pagando.
	if err := s.removeTheGrantsStance(r, row, flag); err != nil {
		return err
	}
	return s.removeConditionalsWithFlag(r, row, flag)
}

// toggleSituational liga ou desliga um condicional de contexto.
func toggleSituational(s Scene, r *http.Request, row sqlcgen.Character, sinais Signals) error {
	chave := ""
	if sinais.Situacao != nil {
		chave = *sinais.Situacao
	}
	return s.plays.ToggleSituational(r.Context(), row.ID, chave)
}

// removeConditionalsWithFlag desliga todo condicional que a postura acendia.
func (s Scene) removeConditionalsWithFlag(r *http.Request, row sqlcgen.Character, flag string) error {
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil || s.deps.Catalogs() == nil {
		return nil
	}
	for _, c := range engine.ComputeItemEffects(s.deps.Catalogs().ActiveItemsFor(ec)).Conditional {
		if c.Flag != flag {
			continue
		}
		_ = s.deps.Queries().RemoveCharacterConditional(r.Context(), sqlcgen.RemoveCharacterConditionalParams{
			Characterid: row.ID, Conditionalid: engine.ConditionalID(c),
		})
	}
	return nil
}
