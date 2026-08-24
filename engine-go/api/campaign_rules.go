package api

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
)

// As regras opcionais da campanha (ALE-221).
//
// O livro não manda aplicar tudo — sobre os limites de carga ele diz "O mestre
// pode ignorar essa regra, desde que os jogadores não abusem" (p141). Aqui mora
// a chave, e ela é UMA: a tela de configuração da campanha e a gaveta de dentro
// da sessão chamam esta mesma rota. Duas portas, um valor.

// campaignRulesDTO é o conjunto das regras DESLIGADAS. Lista e não mapa das
// ligadas, pelo mesmo motivo da tabela: o vazio significa "tudo em vigor", e uma
// regra nova entra valendo sem retro-preencher campanha nenhuma.
type campaignRulesDTO struct {
	IgnoredRules []string `json:"ignoredRules"`
}

// ignoredRulesOf lê o conjunto de uma campanha. Devolve fatia vazia e nunca
// nula: `null` e `[]` chegam diferentes no JSON e a tela teria de tratar os dois.
func (s *Server) ignoredRulesOf(ctx context.Context, campaignID int64) []string {
	rules, err := s.queries.ListIgnoredRulesForCampaign(ctx, campaignID)
	if err != nil || rules == nil {
		return []string{}
	}
	return rules
}

// handleReplaceCampaignRules grava o conjunto INTEIRO das regras desligadas.
//
// Substituição e não delta de propósito: a tela mostra todos os interruptores de
// uma vez, então ela sempre sabe o estado completo, e um par ligar/desligar
// disputando a mesma regra terminaria num estado que ninguém pediu.
func (s *Server) handleReplaceCampaignRules(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		IgnoredRules []string `json:"ignoredRules"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	// A posse vem ANTES da validação do corpo: um estranho que mandasse uma regra
	// inválida aprenderia, pela mensagem, que a campanha existe.
	if _, ok := s.ownedCampaign(w, r, id); !ok {
		return
	}
	wanted, msg := normalizeIgnoredRules(body.IgnoredRules)
	if msg != "" {
		writeValidationError(w, FieldErrorMap{"ignoredRules": {msg}})
		return
	}
	if err := s.gravaRegrasIgnoradas(r.Context(), id, wanted); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update campaign rules")
		return
	}
	writeJSON(w, http.StatusOK, campaignRulesDTO{IgnoredRules: wanted})
}

// normalizeIgnoredRules ordena, tira repetidos e recusa o que o motor não
// conhece. A recusa nomeia o valor e o que era esperado, que é a regra da casa
// para mensagem de exceção — "invalid rule" mandaria o mestre adivinhar.
func normalizeIgnoredRules(raw []string) ([]string, string) {
	seen := map[string]bool{}
	out := []string{}
	for _, rule := range raw {
		if !engine.IsKnownRule(rule) {
			return nil, fmt.Sprintf("unknown rule %q — expected one of %v", rule, engine.KnownRules)
		}
		if seen[rule] {
			continue
		}
		seen[rule] = true
		out = append(out, rule)
	}
	sort.Strings(out)
	return out, ""
}

// gravaRegrasIgnoradas troca o conjunto INTEIRO: limpa e reinsere.
//
// Extraída na ALE-255 porque a cena do servidor precisa do mesmo passo, e é
// substituição e não delta de propósito — o conjunto é pequeno e fechado, e
// mandar o estado final faz a operação ser idempotente. Um delta reenviado
// alternaria a regra duas vezes, que é exatamente o que um clique repetido
// numa conexão ruim produz.
func (s *Server) gravaRegrasIgnoradas(ctx context.Context, campanhaID int64, regras []string) error {
	if err := s.queries.ClearIgnoredRulesForCampaign(ctx, campanhaID); err != nil {
		return err
	}
	agora := nowISO()
	for _, regra := range regras {
		if err := s.queries.IgnoreRuleInCampaign(ctx, sqlcgen.IgnoreRuleInCampaignParams{
			Campaignid: campanhaID, Rule: regra, Updatedat: agora,
		}); err != nil {
			return err
		}
	}
	return nil
}
