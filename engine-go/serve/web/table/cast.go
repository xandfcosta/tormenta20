package table

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/go-chi/chi/v5"

	"t20engine/domain/live"
)

// O ELENCO DA CAMPANHA — o caminho.
//
// UM jogador por vez, que é o buraco que o "Adicionar grupo" não cobre: ele traz
// o grupo INTEIRO, e a cena em que só a Arwen desce na cripta não tem gesto
// nenhum. O combatente inventado à mão para resolver isso nasce sem
// `characterId` — desligado da ficha, sem PV de verdade e fora do descanso.

func (s Scene) CastRoutes(r chi.Router) {
	r.Post(sessionPattern+"/elenco/{characterId}/na-fila",
		s.gmCommand(putPlayerTracker))
	r.Post(sessionPattern+"/elenco/{characterId}/vitais/{pool}/ferir/{step}",
		s.gmCommand(moveCastVitals(-1)))
	r.Post(sessionPattern+"/elenco/{characterId}/vitais/{pool}/curar/{step}",
		s.gmCommand(moveCastVitals(+1)))
}

// putPlayerTracker traz UM personagem do roster para o combate.
//
// Ele passa pelo MESMO `populateParty` do "Adicionar grupo", com uma lista de
// um: é ele que resolve o `characterId`, os vitais e a idempotência. Uma
// segunda forma de pôr PC na fila divergiria no dia em que a primeira mudasse —
// e a diferença apareceria como uma linha sem ficha, que é justamente o defeito
// que este gesto existe para não repetir.
func putPlayerTracker(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	escolhido, err := castMemberOf(st, c)
	if err != nil {
		return nil, err
	}
	estado, err := st.deps.PopulateParty(c.SessionID, []Combatant{*escolhido})
	if estado == nil {
		estado = st.deps.Sessions().GetState(c.SessionID)
	}
	return estado, err
}

// castMemberOf resolve o personagem do CAMINHO contra o roster da campanha.
//
// A TRAVA é esta, e ela é do servidor: o id vem do caminho, e o caminho é
// digitável. Sem a conferência, o mestre de uma mesa alcançaria o personagem de
// OUTRA campanha — poria na fila, e também FERIRIA.
//
// Ela é função própria porque agora tem dois chamadores, e porque uma trava
// copiada é uma trava que diverge no dia em que alguém apertar só uma delas.
func castMemberOf(st Scene, c commandCtx) (*Combatant, error) {
	characterID, err := strconv.ParseInt(chi.URLParam(c.R, "characterId"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("personagem inválido: %q", chi.URLParam(c.R, "characterId"))
	}
	combatentes, err := st.playerCombatants(c.R.Context(), c.CampaignID)
	if err != nil {
		return nil, errors.New("não deu para carregar o grupo desta campanha")
	}
	escolhido := combatantFor(combatentes, characterID)
	if escolhido == nil {
		return nil, fmt.Errorf("o personagem %d não é jogador desta campanha", characterID)
	}
	return escolhido, nil
}

// moveCastVitals fere e cura pelo ELENCO, com o personagem na fila ou fora dela.
//
// Ele existe porque as rotas da fila são por `entryId`, e o elenco é justamente
// onde mora quem não tem linha na iniciativa. A conta e os passos são os mesmos
// — o `poolDeltas` e o `vitalSteps` são compartilhados de propósito, senão as
// duas telas passariam a chamar de "um golpe" coisas diferentes.
func moveCastVitals(sign int64) func(Scene, commandCtx) (*live.SessionRuntimeState, error) {
	return func(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
		raw := chi.URLParam(c.R, "step")
		step, ok := vitalSteps[raw]
		if !ok {
			return nil, fmt.Errorf("passo %q não existe; a tela oferece 1 (clique) e 5 (Shift+clique)", raw)
		}
		pool := chi.URLParam(c.R, "pool")
		hp, mp, ok := poolDeltas(pool, sign*step)
		if !ok {
			return nil, fmt.Errorf("pool %q não existe; o elenco mexe em 'hp' e em 'mp'", pool)
		}
		escolhido, err := castMemberOf(st, c)
		if err != nil {
			return nil, err
		}
		estado, err := st.deps.Sessions().DeltaCharacterVitals(c.SessionID, escolhido.CharacterID, hp, mp)
		// A ficha de quem está na mesa MUDOU, e a tela dele precisa saber —
		// aqui sempre há personagem atrás do gesto, ao contrário da fila,
		// onde o capanga anônimo não tem quem avisar.
		if err == nil {
			st.deps.CharacterChanged(escolhido.CharacterID)
		}
		return estado, err
	}
}

// playerCombatants é todo personagem da campanha com os vitais vivos — o "pôr o
// grupo na fila" de um clique do mestre.
//
// SEM filtro de papel, e isso é a regra e não um esquecimento: o mestre não tem
// personagem próprio na maioria das mesas, os NPCs dele não são membros da
// campanha, e a coluna `role` nunca teve outro valor além de `player` em
// produção. Filtrar aqui esconderia da fila o bardo que o mestre também joga.
func (s Scene) playerCombatants(ctx context.Context, campaignID int64) ([]Combatant, error) {
	linhas, err := s.deps.Queries().ListMembers(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	grupo := make([]Combatant, 0, len(linhas))
	for _, m := range linhas {
		grupo = append(grupo, Combatant{
			CharacterID: m.Characterid, Name: m.Charname,
			HpCurrent: m.Charhpcurrent, HpMax: m.Charhpmax,
			MpCurrent: m.Charmpcurrent, MpMax: m.Charmpmax,
		})
	}
	return grupo, nil
}

// combatantFor acha o combatente do personagem pedido. Nil é a resposta para
// "não é jogador desta campanha", e quem chama decide o que fazer com isso.
func combatantFor(combatentes []Combatant, characterID int64) *Combatant {
	for i := range combatentes {
		if combatentes[i].CharacterID == characterID {
			return &combatentes[i]
		}
	}
	return nil
}

// ── as expressões que o Datastar executa ────────────────────────────────────

func openSheetCast(m Member) string {
	return fmt.Sprintf("document.getElementById('elenco-%d').showModal()", m.CharacterID)
}

func closeSheetCast(m Member) string {
	return fmt.Sprintf("document.getElementById('elenco-%d').close();", m.CharacterID)
}

// poeNaFila fecha ANTES de postar, e isso não é arrumação: a recusa do servidor
// acende no `$command_error`, que é do RODAPÉ — um diálogo aberto por cima dela
// esconderia a única frase que explica o que houve.
func poeNaFila(v View, m Member) string {
	return fmt.Sprintf("@post('/campanhas/%d/sessoes/%d/elenco/%d/na-fila')",
		v.CampaignID, v.SessionID, m.CharacterID)
}

// castVital escreve o gesto com os DOIS passos já resolvidos em duas URLs, e o
// `evt.shiftKey` escolhendo entre elas — a mesma forma do `rowVital` da fila, e
// pela mesma razão: o número nunca viaja como dado.
func castVital(v View, m Member, pool, verb string) string {
	base := fmt.Sprintf("/campanhas/%d/sessoes/%d/elenco/%d/vitais/%s/%s/",
		v.CampaignID, v.SessionID, m.CharacterID, pool, verb)
	return fmt.Sprintf("@post(evt.shiftKey ? '%s5' : '%s1')", base, base)
}
