package sheetui

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"t20engine/infra/db/sqlcgen"
)

// OS GESTOS DA SUPERFÍCIE AÇÕES que terminam na MESA.
//
// Nenhum deles decide nada: o que é regra — de quem é a vez, se a ação padrão
// sobrou, se a arma existe, se o alvo está na fila — mora no `app/combat`, e
// quem chega até lá é a porta. O que mora aqui é o que é da TELA: de onde sai o
// alvo (o sinal que a barra da Mesa escreve) e de onde sai a arma (o índice no
// caminho).

// rollsAttackOnTheTarget rola o golpe da vez contra o alvo escolhido.
func rollsAttackOnTheTarget(s Scene, r *http.Request, row sqlcgen.Character, sig Signals) error {
	weapon, err := strconv.Atoi(chi.URLParam(r, "arma"))
	if err != nil {
		return fmt.Errorf("a arma vem por índice no caminho, e veio %q", chi.URLParam(r, "arma"))
	}
	return s.proposeOnTheTable(r, row, ActionStrike{Weapon: weapon}, sig)
}

// rollsManeuverOnTheTarget rola a manobra da vez contra o alvo escolhido
// (p234).
//
// A ARMA É A PRIMEIRA e não vem no caminho: uma manobra não causa dano, e o que
// o motor pede dela é só a conferência de que quem a faz não está de arco na
// mão. Um seletor de arma por manobra seriam cinco escolhas por nada.
func rollsManeuverOnTheTarget(s Scene, r *http.Request, row sqlcgen.Character, sig Signals) error {
	return s.proposeOnTheTable(r, row, ActionStrike{Maneuver: chi.URLParam(r, "manobra")}, sig)
}

// proposeOnTheTable é o que os dois gestos têm em comum: o alvo.
//
// A RECUSA DO ALVO VAZIO É DAQUI e não do motor, e é a única decisão que esta
// camada toma: "escolha um alvo" é uma frase sobre a TELA — sobre uma barra que
// está logo acima e que a pessoa não usou —, e o `app/combat` não sabe que essa
// barra existe. O botão já nasce desabilitado sem alvo; isto é o que sobra para
// quem chegou pelo endereço.
func (s Scene) proposeOnTheTable(
	r *http.Request, row sqlcgen.Character, strike ActionStrike, sig Signals,
) error {
	strike.TargetEntryID = strings.TrimSpace(sig.TurnTarget)
	if strike.TargetEntryID == "" {
		return fmt.Errorf("escolha o alvo da vez na barra acima antes de rolar")
	}
	return s.deps.ProposeStrikeOnTable(r.Context(), row.ID, strike)
}
