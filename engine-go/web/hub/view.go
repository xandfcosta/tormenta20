package hub

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode"
)

// O HUB como dado (ALE-231) — o menu principal do jogo.
//
// Ele é a terceira superfície do Datastar e a primeira com CROMO DE JOGO:
// navegação por setas, cues de áudio, tela cheia e um diálogo com formulário.
// Se o modelo não sustentasse isso, era melhor descobrir aqui do que no
// tabuleiro.

type hubView struct {
	Nome string
	// Inicial é a letra do retrato. Calculada aqui e não no template porque é
	// regra de apresentação (maiúscula, e "?" quando não sobra letra), e regra
	// em template é regra escondida onde ninguém a testa.
	Inicial string
	EhAdmin bool
	// Viva é a sessão que "Continuar sessão" retoma, ou nil. Uma consulta, não
	// N+1 — ver `sessaoViva`.
	Viva *hubSessaoViva
}

type hubSessaoViva struct {
	CampaignID int64
	SessionID  int64
	// Rota é PARA ONDE o "Continuar sessão" leva, resolvido por quem hospeda a
	// cena (`Deps.MesaRoute`). Ela é campo da view e não uma chamada no
	// template pela mesma razão da `Inicial` acima.
	Rota string
}

// carregaHub monta a tela inteira.
func (s Scene) carregaHub(ctx context.Context, eu Viewer) (hubView, error) {
	viva, err := s.sessaoViva(ctx, eu.ID)
	if err != nil {
		return hubView{}, err
	}
	nome := nomeDeExibicao(eu)
	return hubView{
		Nome:    nome,
		Inicial: inicialDe(nome),
		EhAdmin: eu.IsAdmin,
		Viva:    viva,
	}, nil
}

// sessaoViva responde "há partida em andamento?" numa consulta só.
//
// Na SPA isto eram 62 linhas de cliente que abriam N+1 requisições — uma lista
// de sessões por campanha —, e o arquivo dizia por quê: não existia rota que
// respondesse a pergunta. Some junto o `createLiveSessionPrefetch`, que existia
// só para esconder a latência que a própria fan-out criava.
//
// "Nenhuma linha" é resposta NORMAL e não erro: quase sempre não há partida
// rolando.
func (s Scene) sessaoViva(ctx context.Context, userID int64) (*hubSessaoViva, error) {
	linha, err := s.deps.Queries().FirstLiveSessionForUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &hubSessaoViva{
		CampaignID: linha.Campaignid,
		SessionID:  linha.Sessionid,
		Rota:       s.deps.MesaRoute(linha.Campaignid, linha.Sessionid),
	}, nil
}

// nomeDeExibicao: o nome quando existe, senão o e-mail, senão "Aventureiro".
// Mesma cadeia da SPA — quem entra por convite pode não ter dado nome nenhum.
func nomeDeExibicao(eu Viewer) string {
	if eu.Name != nil && strings.TrimSpace(*eu.Name) != "" {
		return strings.TrimSpace(*eu.Name)
	}
	if eu.Email != "" {
		return eu.Email
	}
	return "Aventureiro"
}

// inicialDe é a letra do retrato, em maiúscula.
//
// Por runa e não por byte: "Ãurea" começa com dois bytes, e cortar o primeiro
// produziria meio caractere — que o navegador desenha como o losango de erro.
func inicialDe(nome string) string {
	for _, r := range strings.TrimSpace(nome) {
		return string(unicode.ToUpper(r))
	}
	return "?"
}
