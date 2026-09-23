package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// A API JSON tem DOIS GRUPOS, e eles respondem a perguntas diferentes.
//
// O `público` é o que existe para quem está fora do repositório. A `bancada` é o
// que existe para a suíte de Playwright montar fixture em duas chamadas em vez de
// seis telas — e ela é deliberada, porque montar tudo pela tela troca segundos
// por minutos.
//
// # Por que os dois grupos, se o endereço de cada rota não mudou
//
// Porque misturados eles fazem a API parecer MAIOR do que é, e essa aparência é o
// que trava a decisão: quem for desenhar uma API de verdade (ALE-343) precisa
// saber o que já tem consumidor de fora — a resposta é UMA rota — e o que só tem
// o e2e. Com os dois grupos, mover a bancada para outro prefixo, ou não montá-la
// em produção, passa a ser uma linha; e nenhuma das quatro respostas possíveis
// daquela issue fica travada por isto.
//
// Quem impede que a separação vire enfeite é o
// `TestEveryJSONRouteDeclaresWhichSideItIsOn`: ele PERCORRE o roteador e falha com
// o endereço de qualquer rota que não esteja declarada num dos dois lados.

// Router é a API JSON, e as rotas daqui NÃO carregam o prefixo `/api`: quem o põe
// é o `web_router`.
//
// Nenhuma cena a chama — as cenas leem o banco pelo `Queries` da porta delas e
// desenham HTML.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	// Sem origem configurada, NENHUM middleware de CORS — que é produção: o
	// binário serve as cenas, então toda chamada é da mesma origem. Montá-lo com
	// `[]string{""}` negaria as mesmas requisições, mas por acidente; e este
	// guarda mantém um `CORS_ORIGIN` vazio longe do padrão do go-chi para lista
	// vazia, que é liberar TODA origem — com credenciais ligadas, todo site.
	if len(s.cfg.CORSOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   s.cfg.CORSOrigins,
			AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: true,
		}))
	}
	s.publicJSONRoutes(r)
	s.benchJSONRoutes(r)
	return r
}

// publicJSONRoutes é a superfície que existe para quem está FORA, e hoje ela é
// uma rota.
//
// `/health` é INFRAESTRUTURA — quem bate nele é o `healthcheck` do compose e o
// `-health` do próprio binário. É o contra-exemplo da faxina de rotas: "rota sem
// consumidor" se decide perguntando quem pergunta DE FORA, e não quem chama de
// dentro do repositório.
func (s *Server) publicJSONRoutes(r chi.Router) {
	r.Get("/health", s.handleHealth)
}

// benchJSONRoutes é a BANCADA do e2e: tudo aqui existe para um spec, e cada rota
// diz qual.
//
// Elas são autenticadas e autorizadas como qualquer outra — a bancada não é uma
// porta dos fundos, é um atalho de montagem. O que ela economiza está medido na
// ALE-343: segundos de setup em vez de minutos.
func (s *Server) benchJSONRoutes(r chi.Router) {
	r.Route("/campanhas", func(r chi.Router) {
		r.Use(s.requireAuth)
		// A varredura do `auth.setup.ts`: lista, filtra pelo prefixo "E2E
		// Descartável" e apaga. Nomeia pelo PREFIXO e nunca por id — apagar por
		// id seria apagar seed.
		r.Get("/", s.handleListCampaigns)
		r.Delete("/{id}", s.handleDeleteCampaign)
		// A fixture do `board.spec.ts`: uma mesa descartável por corrida, montada
		// em duas chamadas em vez de seis telas.
		r.Post("/", s.handleCreateCampaign)
		r.Route("/{campaignId}/sessoes", func(r chi.Router) {
			r.Post("/", s.handleCreateSession)
		})
	})

	r.Route("/personagens", func(r chi.Router) {
		r.Use(s.requireAuth)
		r.Use(s.serializeCharacterWrites)
		// A varredura das CONDIÇÕES: o spec da sessão aplica Abalado, Agarrado e
		// Cego para medir a faixa cheia, e a limpeza dele mora no corpo do teste
		// — quando ele falha, a condição fica gravada e ele falha PARA SEMPRE.
		r.Get("/", s.handleListCharacters)
		r.Patch("/{id}/conditions", s.handleUpdateConditions)
	})
}
