package api

import (
	"context"
	"t20engine/plataforma"

	"t20engine/db/sqlcgen"
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

// A normalização das regras opcionais mora em `campaign` desde a ALE-278: ela é
// pura (só pergunta ao motor se a regra existe) e a cena precisa dela.

// saveIgnoredRules troca o conjunto INTEIRO: limpa e reinsere.
//
// Extraída na ALE-255 porque a cena do servidor precisa do mesmo passo, e é
// substituição e não delta de propósito — o conjunto é pequeno e fechado, e
// mandar o estado final faz a operação ser idempotente. Um delta reenviado
// alternaria a regra duas vezes, que é exatamente o que um clique repetido
// numa conexão ruim produz.
func (s *Server) saveIgnoredRules(ctx context.Context, campanhaID int64, regras []string) error {
	if err := s.queries.ClearIgnoredRulesForCampaign(ctx, campanhaID); err != nil {
		return err
	}
	agora := plataforma.NowISO()
	for _, regra := range regras {
		if err := s.queries.IgnoreRuleInCampaign(ctx, sqlcgen.IgnoreRuleInCampaignParams{
			Campaignid: campanhaID, Rule: regra, Updatedat: agora,
		}); err != nil {
			return err
		}
	}
	return nil
}
