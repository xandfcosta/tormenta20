package api

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"t20engine/plataforma"
	"t20engine/sheet"
	"t20engine/web/ui"
)

// A CARTA DE CONVITE como dado (ALE-249): entrar numa mesa.
//
// Duas portas para a mesma sala, e é isso que a struct precisa representar: um
// LINK de convite (`?token=…`, que já diz qual é a mesa) ou o NÚMERO que o
// mestre leu em voz alta. Elas são exclusivas na tela — quem chegou pelo link
// não digita número —, e por isso o `TemConvite` decide qual metade aparece.
//
// O ganho desta cena sobre a da SPA é o convite ser resolvido AQUI. Lá a tela
// monta, dispara `GET /invites/{token}` e mostra um esqueleto enquanto espera;
// aqui o nome da campanha já vem na primeira resposta. Some o estado de
// "carregando" inteiro — não porque foi escondido, porque não existe.
type campanhaEntrarView struct {
	Convite     string
	TemConvite  bool
	ConviteVale bool
	// NomeDaCampanha só vale quando o convite vale. É o que diz à pessoa PARA
	// QUAL mesa ela foi chamada antes de comprometer um herói com ela.
	NomeDaCampanha string
	CampanhaID     int64
	// NumeroDigitado volta preenchido numa recusa, como todo campo desta casa.
	NumeroDigitado string
	Herois         []heroiDaCarta
	EscolhidoID    int64
	Erros          plataforma.FieldErrorMap
	Aviso          string
}

// heroiDaCarta é uma plaqueta escolhível: o mínimo para reconhecer o herói.
type heroiDaCarta struct {
	ID        int64
	Nome      string
	Subtitulo string
	Iniciais  string
	Gradiente string
}

func (s *Server) carregaCartaDeConvite(ctx context.Context, eu AuthUser, token string) (campanhaEntrarView, error) {
	v := campanhaEntrarView{Convite: token, TemConvite: token != "", Erros: plataforma.FieldErrorMap{}}

	if v.TemConvite {
		// Convite morto NÃO é erro da página: é uma resposta, e a carta diz
		// isso em voz alta para a pessoa pedir outro link em vez de ficar
		// olhando um botão que não envia (ALE-80).
		c, err := s.queries.GetCampaignByToken(ctx, sql.NullString{String: token, Valid: true})
		switch {
		case err == nil:
			v.ConviteVale, v.NomeDaCampanha, v.CampanhaID = true, c.Name, c.ID
		case errors.Is(err, sql.ErrNoRows):
			// deixa `ConviteVale` falso — a carta mostra a recusa
		default:
			return campanhaEntrarView{}, err
		}
	}

	elenco, err := s.characterList(ctx, eu.ID)
	if err != nil {
		return campanhaEntrarView{}, err
	}
	for _, c := range elenco {
		v.Herois = append(v.Herois, heroiDaCarta{
			ID: c.ID, Nome: c.Name,
			Subtitulo: subtituloDoHeroi(c),
			Iniciais:  ui.Monogram(c.Name),
			Gradiente: ui.NameGradient(c.Name),
		})
	}
	return v, nil
}

// subtituloDoHeroi é a linha de baixo da plaqueta: as classes com nível, ou o
// nível sozinho para quem ainda não tem classe. É o `classLevelLine` da SPA com
// o mesmo recuo.
func subtituloDoHeroi(c sheet.CharacterDTO) string {
	if linha := classesDoHeroi(c); linha != "" {
		return linha
	}
	return "Nv " + strconv.FormatInt(c.Level, 10)
}
