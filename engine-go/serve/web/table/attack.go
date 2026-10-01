package table

import (
	"fmt"

	"github.com/go-chi/chi/v5"

	"t20engine/app"
	"t20engine/app/combat"
	"t20engine/domain/board"
	"t20engine/domain/live"
	"t20engine/serve/web/routes"
)

// O GESTO DE ATACAR, e a divisa dele é a do movimento: propor é de quem joga,
// confirmar é de quem mestra.
//
// O ALVO vem no caminho e QUEM ATACA é a vez. É o que o gesto de menu permite
// dizer — o menu abre sobre o alvo, e não há segundo clique onde caberia
// escolher o atacante —, e é também o que o livro diz: combate acontece em
// turnos, e quem age é quem está na vez (p231).
func (s Scene) AttackRoutes(r chi.Router) {
	r.Post(sessionPattern+"/iniciativa/{entryId}/atacar", s.tableStateCommand(proposesAttack))
	// A MANOBRA divide os verbos de confirmar e cancelar com o golpe, e só a
	// proposta é dela: uma manobra É um ataque corpo a corpo (p234), e o
	// provisório é o mesmo.
	r.Post(sessionPattern+"/iniciativa/{entryId}/manobra/{kind}", s.tableStateCommand(proposesManeuver))
	// O ATAQUE A OBJETO tem rota própria porque o ALVO é de outra espécie: uma
	// peça e não uma linha da fila. Mesmo verbo, mesmo prefixo de tabuleiro que
	// os outros gestos de peça.
	r.Post(sessionPattern+"/tabuleiro/pecas/{tokenId}/atacar", s.tableStateCommand(proposesAttackOnObject))
	r.Post(sessionPattern+"/ataque/confirmar", s.gmCommand(confirmsAttack))
	r.Post(sessionPattern+"/ataque/cancelar", s.tableStateCommand(cancelsAttack))
}

// proposesAttack rola o ataque de quem está na vez contra a linha do caminho.
func proposesAttack(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	return st.proposeStrike(c, "")
}

// proposeStrike é o que o golpe e a manobra têm em comum, que é tudo menos a
// regra: a vez, a posse conferida contra o BANCO, e o provisório.
//
// A manobra vazia é o golpe. Um ramo aqui, e não dois caminhos: as sete
// conferências antes da rolagem são as mesmas, e duplicá-las faria a próxima
// correção acertar uma e esquecer a outra.
func (st Scene) proposeStrike(c commandCtx, maneuver string) (*live.SessionRuntimeState, error) {
	state, err := st.deps.Sessions().State(c.R.Context(), c.SessionID)
	if err != nil {
		return nil, err
	}
	if state == nil || state.TurnIndex < 0 || state.TurnIndex >= len(state.Initiative) {
		return nil, fmt.Errorf("fora de combate não há vez, e é a vez que diz quem ataca")
	}
	onTurn := state.Initiative[state.TurnIndex]
	// A POSSE é resolvida CONTRA O BANCO e nunca contra o cliente — é o mesmo
	// caminho que o `Mover.OwnsCharacter` do tabuleiro usa.
	_, mine, _ := st.tableRoster(c.R.Context(), c.User, c.CampaignID)
	if _, err := st.strike.Propose(c.R.Context(), app.Caller{ID: c.User}, c.Role, combat.Request{
		CampaignID:      c.CampaignID,
		SessionID:       c.SessionID,
		AttackerEntryID: onTurn.ID,
		TargetEntryID:   chi.URLParam(c.R, "entryId"),
		OwnsAttacker:    onTurn.CharacterID != nil && mine[*onTurn.CharacterID],
		Maneuver:        maneuver,
	}); err != nil {
		return nil, err
	}
	return st.deps.Sessions().State(c.R.Context(), c.SessionID)
}

// proposesManeuver rola a manobra de quem está na vez contra a linha do caminho.
//
// Ele divide com o golpe a conferência inteira — a vez, a posse, a ação padrão —
// e muda só a REGRA no meio, que é o ramo do `Strike`. A MANOBRA vem no caminho
// e não no corpo porque o gesto é de menu: o mestre escolhe "Derrubar" num item,
// e um corpo JSON exigiria uma ilha de JS onde há um `@post`.
func proposesManeuver(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	return st.proposeStrike(c, chi.URLParam(c.R, "kind"))
}

// proposesAttackOnObject rola o golpe de quem está na vez contra uma PEÇA DE
// CENÁRIO (p239).
//
// A PEÇA É LIDA AQUI e não no `combat`, e essa é a divisa: o tabuleiro é um
// agregado que o caso de uso de ataque não importa — ele recebe tamanho e
// material e pergunta ao MOTOR a Defesa e a RD. É a mesma razão do
// `BoardSituations`, que já traduz o mapa em regra antes de atravessar.
func proposesAttackOnObject(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	state, err := st.deps.Sessions().State(c.R.Context(), c.SessionID)
	if err != nil {
		return nil, err
	}
	if state == nil || state.TurnIndex < 0 || state.TurnIndex >= len(state.Initiative) {
		return nil, fmt.Errorf("fora de combate não há vez, e é a vez que diz quem ataca")
	}
	onTurn := state.Initiative[state.TurnIndex]
	tokenID := chi.URLParam(c.R, "tokenId")
	alvo, err := st.objectTokenOf(c, tokenID)
	if err != nil {
		return nil, err
	}
	_, mine, _ := st.tableRoster(c.R.Context(), c.User, c.CampaignID)
	if _, err := st.strike.Propose(c.R.Context(), app.Caller{ID: c.User}, c.Role, combat.Request{
		CampaignID:      c.CampaignID,
		SessionID:       c.SessionID,
		AttackerEntryID: onTurn.ID,
		TargetObject:    alvo,
		OwnsAttacker:    onTurn.CharacterID != nil && mine[*onTurn.CharacterID],
	}); err != nil {
		return nil, err
	}
	return st.deps.Sessions().State(c.R.Context(), c.SessionID)
}

// objectTokenOf acha a peça e RECUSA a que não tem estatísticas.
//
// A recusa é pelo nome e diz o que falta: "cenário que não se ataca" é um estado
// legítimo — a mancha de musgo —, e um erro genérico aqui faria o mestre
// procurar defeito no gesto em vez de no PV que ele não deu à peça.
func (st Scene) objectTokenOf(c commandCtx, tokenID string) (*combat.ObjectTarget, error) {
	b, err := st.deps.Boards().Get(c.R.Context(), c.SessionID, c.BoardID)
	if err != nil {
		return nil, err
	}
	peca := board.FindToken(b, tokenID)
	if peca == nil {
		return nil, fmt.Errorf("a peça %q não está no tabuleiro: %w", tokenID, app.ErrNotFound)
	}
	if !peca.HasObjectStats() {
		return nil, fmt.Errorf(
			"%s é cenário sem PV e não se ataca: dê tamanho, material e PV a ela (p239)", peca.Label)
	}
	if peca.IsDestroyed() {
		return nil, fmt.Errorf("%s já foi destruída", peca.Label)
	}
	return &combat.ObjectTarget{
		TokenID: peca.ID, Label: peca.Label, Size: peca.Size, Material: peca.Material,
	}, nil
}

// confirmsAttack liquida o provisório, e é ele quem ESCOLHE onde o dano pousa.
//
// DOIS AGREGADOS, e a cena é o único lugar acima dos dois: o PV de uma criatura
// mora na fila e o de um objeto mora no tabuleiro, cada store com trava própria.
// Um chamando o outro de dentro da sua é como se escreve um abraço mortal, e o
// `boards.Store` diz isso por escrito.
//
// A ORDEM É A MESMA DO `CommitAttack`: o dano primeiro, o provisório depois.
// Invertida, uma falha ao gravar o PV deixaria a mesa sem o provisório e sem o
// dano — e ninguém saberia que o ataque existiu.
func confirmsAttack(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	pendente, err := pendingAttackOf(st, c)
	if err != nil {
		return nil, err
	}
	if pendente != nil && pendente.TargetTokenID != "" && pendente.Damage > 0 {
		if _, err := st.deps.Boards().DamageObject(c.R.Context(), c.SessionID, c.BoardID,
			pendente.TargetTokenID, pendente.Damage); err != nil {
			return nil, err
		}
	}
	return st.deps.Sessions().CommitAttack(c.R.Context(), c.SessionID, live.Attacker{UserID: c.User, Role: c.Role})
}

// pendingAttackOf devolve o provisório em jogo, ou nulo quando não há.
//
// Nulo e não erro: quem recusa "não há ataque para confirmar" é o
// `CommitAttack`, com a frase dele — duplicar a recusa aqui daria duas
// mensagens para o mesmo não.
func pendingAttackOf(st Scene, c commandCtx) (*live.PendingAttack, error) {
	state, err := st.deps.Sessions().State(c.R.Context(), c.SessionID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, nil
	}
	return state.PendingAttack, nil
}

func cancelsAttack(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	return st.deps.Sessions().CancelAttack(c.R.Context(), c.SessionID, live.Attacker{UserID: c.User, Role: c.Role})
}

// attackCommand escreve o `@post` dos dois verbos da faixa.
func attackCommand(v View, verb string) string {
	return fmt.Sprintf("@post('%s/ataque/%s')", routes.Session(v.CampaignID, v.SessionID), verb)
}

// attackOnObject é o `@post` do ataque a uma PEÇA DE CENÁRIO: a peça no caminho,
// e quem ataca é a vez. Irmão do `attackOnTarget`, com a rota do tabuleiro.
func attackOnObject(v BoardView, tokenID string) string {
	return fmt.Sprintf("@post('%s/tabuleiro/pecas/%s/atacar')",
		routes.Session(v.CampaignID, v.SessionID), tokenID)
}

// movingLabel é o que o botão de "em movimento" diz, nos dois estados.
//
// DOIS TEXTOS e não um: um rótulo fixo ("Em movimento") num botão de alternar
// não diz se o clique LIGA ou DESLIGA, e o `aria-pressed` sozinho só alcança
// quem usa leitor de tela.
func movingLabel(p boardToken) string {
	if p.Moving {
		return "Parar " + p.Label + " — ela deixa de receber +5 na Defesa"
	}
	return p.Label + " em movimento — +5 na Defesa (p239)"
}

// attackOnTarget é o `@post` do menu da peça: o ALVO no caminho, e quem ataca é
// a vez.
func attackOnTarget(v BoardView, entryID string) string {
	return fmt.Sprintf("@post('%s/iniciativa/%s/atacar')",
		routes.Session(v.CampaignID, v.SessionID), entryID)
}

// maneuverOnTarget é o `@post` de uma manobra do menu da peça, com a manobra no
// CAMINHO — o gesto é de menu, e um corpo JSON pediria uma ilha de JS onde há um
// `@post`.
//
// Ele fecha as DUAS camadas, como o `copyCommand`: escolher a manobra fecha o
// submenu e o menu da peça, e o popover não se fecha sozinho quando o clique é
// num botão dentro dele.
func maneuverOnTarget(v BoardView, tokenID, entryID, kind string) string {
	return closesTheManeuverMenu(tokenID) +
		fmt.Sprintf("@post('%s/iniciativa/%s/manobra/%s')",
			routes.Session(v.CampaignID, v.SessionID), entryID, kind) +
		"; " + closeMenuToken
}
