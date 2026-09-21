package hub

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode"
)

// O HUB como dado — o menu principal do jogo, com o CROMO DE JOGO da casa:
// navegação por setas, cues de áudio, tela cheia e um diálogo com formulário.

type hubView struct {
	Nome string
	// Inicial é a letra do retrato. Calculada aqui e não no template porque é
	// regra de apresentação (maiúscula, e "?" quando não sobra letra), e regra
	// em template é regra escondida onde ninguém a testa.
	Inicial string
	EhAdmin bool
	// Viva é a sessão que "Continuar sessão" retoma, ou nil. Uma consulta, não
	// N+1 — ver `liveSession`.
	Viva *hubLiveSession
}

type hubLiveSession struct {
	CampaignID int64
	SessionID  int64
	// Rota é PARA ONDE o "Continuar sessão" leva, resolvido por quem hospeda a
	// cena (`Deps.MesaRoute`). Ela é campo da view e não uma chamada no
	// template pela mesma razão da `Inicial` acima.
	Rota string
}

// loadHub monta a tela inteira.
func (s Scene) loadHub(ctx context.Context, eu Viewer) (hubView, error) {
	viva, err := s.liveSession(ctx, eu.ID)
	if err != nil {
		return hubView{}, err
	}
	nome := displayName(eu)
	return hubView{
		Nome:    nome,
		Inicial: initialOf(nome),
		EhAdmin: eu.IsAdmin,
		Viva:    viva,
	}, nil
}

// liveSession responde "há partida em andamento?" numa consulta só.
//
// UMA consulta e não uma lista de sessões por campanha: a fan-out abre N+1
// requisições e depois cobra um prefetch só para esconder a latência que ela
// mesma criou.
//
// "Nenhuma linha" é resposta NORMAL e não erro: quase sempre não há partida
// rolando.
func (s Scene) liveSession(ctx context.Context, userID int64) (*hubLiveSession, error) {
	linha, err := s.deps.Queries().FirstLiveSessionForUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &hubLiveSession{
		CampaignID: linha.Campaignid,
		SessionID:  linha.Sessionid,
		Rota:       s.deps.TableRoute(linha.Campaignid, linha.Sessionid),
	}, nil
}

// displayName: o nome quando existe, senão o e-mail, senão "Aventureiro" —
// quem entra por convite pode não ter dado nome nenhum.
func displayName(eu Viewer) string {
	if eu.Name != nil && strings.TrimSpace(*eu.Name) != "" {
		return strings.TrimSpace(*eu.Name)
	}
	if eu.Email != "" {
		return eu.Email
	}
	return "Aventureiro"
}

// initialOf é a letra do retrato, em maiúscula.
//
// Por runa e não por byte: "Ãurea" começa com dois bytes, e cortar o primeiro
// produziria meio caractere — que o navegador desenha como o losango de erro.
func initialOf(nome string) string {
	for _, r := range strings.TrimSpace(nome) {
		return string(unicode.ToUpper(r))
	}
	return "?"
}
