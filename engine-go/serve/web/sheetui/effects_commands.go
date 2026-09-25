package sheetui

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

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
// Encerrar apaga a linha da postura, desliga os condicionais e leva embora o que
// ela concedeu — as três numa transação, no caso de uso. Entrar continua sendo
// dos Poderes, onde o PM é cobrado; aqui não há como pagar nada.
func endStance(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	return s.plays.EndStance(r.Context(), row, dto, chi.URLParam(r, "flag"))
}

// toggleSituational liga ou desliga um condicional de contexto.
func toggleSituational(s Scene, r *http.Request, row sqlcgen.Character, signals Signals) error {
	key := ""
	if signals.Status != nil {
		key = *signals.Status
	}
	// O AGREGADO vai junto porque quem decide se a chave vale é o caso de uso, e
	// ele precisa saber o que ESTA ficha oferece — a tela esconder não basta.
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	return s.plays.ToggleSituational(r.Context(), dto, key)
}
