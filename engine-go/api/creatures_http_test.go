package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

// O bloco do Bandido como o livro imprime (p289). Serve de fixture porque é um
// bloco REAL e completo: tem perícia, equipamento e tesouro, que são justo os
// campos que a nossa importação do bestiário perdeu (ALE-151).
const blocoBandido = `{
  "nd": 0.25, "tipo": "humanoide", "size": "medio",
  "iniciativa": 4, "percepcao": 1,
  "defesa": 13, "fortitude": 1, "reflexos": 3, "vontade": -1,
  "hp": 6, "deslocamento": "9m (6q)",
  "forca": 1, "destreza": 2, "constituicao": 1,
  "inteligencia": 0, "sabedoria": -1, "carisma": 0,
  "attacks": [{"name": "Clava", "attackBonus": 7, "damage": "1d6+3"}],
  "skills": [{"name": "Furtividade", "bonus": 5}],
  "equipment": "Clava", "treasure": "Metade",
  "specialAbilities": []
}`

func corpoCriatura(nome, bloco string) string {
	return `{"name":` + strconv.Quote(nome) + `,"block":` + bloco + `}`
}

// A regra de PAPEL é a razão de esta família existir no servidor e não só na
// tela: o bloco é informação do MESTRE, e o jogador continua vendo nome e barra
// de PV pela iniciativa (decisão do dono na ALE-137). Esconder o botão é UX; o
// limite é aqui.
func TestOnlyTheGmReadsAndWritesACreature(t *testing.T) {
	s := newTestServer(t)
	mestre := seedUser(t, s, "mestre@t20.local")
	jogador := seedUser(t, s, "jogador@t20.local")
	campanha := seedCampaign(t, s, mestre)
	// Vínculo de jogador é por PERSONAGEM: `roleIn` procura membro cujo
	// personagem tem este dono.
	seedMember(t, s, campanha, seedCharacter(t, s, jogador, "Herói", 10, 10, 0, 0), "player")

	criada := authed(t, s, mestre, http.MethodPost,
		"/campaigns/"+id64(campanha)+"/creatures", corpoCriatura("Bandido", blocoBandido))
	if criada.Code != http.StatusCreated {
		t.Fatalf("mestre criando: %d — %s", criada.Code, criada.Body.String())
	}

	lista := authed(t, s, jogador, http.MethodGet, "/campaigns/"+id64(campanha)+"/creatures", "")
	if lista.Code != http.StatusForbidden {
		t.Errorf("jogador lendo a lista: %d, queria 403", lista.Code)
	}
	escrita := authed(t, s, jogador, http.MethodPost,
		"/campaigns/"+id64(campanha)+"/creatures", corpoCriatura("Espião", blocoBandido))
	if escrita.Code != http.StatusForbidden {
		t.Errorf("jogador criando: %d, queria 403", escrita.Code)
	}
}

// O bloco volta INTEIRO. Um campo que some no vai-e-vem é pior que um campo que
// nunca existiu: o mestre digita o equipamento do vilão, a tela confirma, e a
// informação se perde no caminho — que é exatamente a queixa que abriu a issue.
func TestACreatureComesBackWithItsWholeBlock(t *testing.T) {
	s := newTestServer(t)
	mestre := seedUser(t, s, "mestre@t20.local")
	campanha := seedCampaign(t, s, mestre)

	criada := authed(t, s, mestre, http.MethodPost,
		"/campaigns/"+id64(campanha)+"/creatures", corpoCriatura("Bandido", blocoBandido))
	if criada.Code != http.StatusCreated {
		t.Fatalf("criar: %d — %s", criada.Code, criada.Body.String())
	}

	var dto creatureDTO
	if err := json.Unmarshal(criada.Body.Bytes(), &dto); err != nil {
		t.Fatalf("resposta ilegível: %v", err)
	}
	if dto.Name != "Bandido" || dto.Block.HP != 6 || dto.Block.Defesa != 13 {
		t.Fatalf("identidade/numeros: %+v", dto)
	}
	if dto.Block.Equipment != "Clava" || dto.Block.Treasure != "Metade" {
		t.Errorf("equipamento/tesouro perdidos: %q / %q", dto.Block.Equipment, dto.Block.Treasure)
	}
	if len(dto.Block.Skills) != 1 || dto.Block.Skills[0].Name != "Furtividade" || dto.Block.Skills[0].Bonus != 5 {
		t.Errorf("perícias perdidas: %+v", dto.Block.Skills)
	}
	if len(dto.Block.Attacks) != 1 || dto.Block.Attacks[0].AttackBonus != 7 {
		t.Errorf("ataques perdidos: %+v", dto.Block.Attacks)
	}
	if dto.Block.Iniciativa != 4 || dto.Block.Percepcao != 1 {
		t.Errorf("iniciativa/percepção perdidas: %d / %d", dto.Block.Iniciativa, dto.Block.Percepcao)
	}
}

// PM é ponteiro porque a linha só existe em conjurador: o Bandido não tem, o
// Centauro Xamã tem 20 (p290). Um zero fixo diria "tem mana e está sem".
func TestACreatureWithoutMpDoesNotInventMp(t *testing.T) {
	s := newTestServer(t)
	mestre := seedUser(t, s, "mestre@t20.local")
	campanha := seedCampaign(t, s, mestre)

	semPM := authed(t, s, mestre, http.MethodPost,
		"/campaigns/"+id64(campanha)+"/creatures", corpoCriatura("Bandido", blocoBandido))
	var bandido creatureDTO
	_ = json.Unmarshal(semPM.Body.Bytes(), &bandido)
	if bandido.Block.PM != nil {
		t.Errorf("bandido veio com PM %d", *bandido.Block.PM)
	}

	comPM := authed(t, s, mestre, http.MethodPost,
		"/campaigns/"+id64(campanha)+"/creatures",
		corpoCriatura("Centauro Xamã", `{"nd":3,"tipo":"humanoide","size":"grande","hp":35,"pm":20,
			"defesa":21,"fortitude":9,"reflexos":4,"vontade":15,"deslocamento":"12m (8q)"}`))
	var xama creatureDTO
	_ = json.Unmarshal(comPM.Body.Bytes(), &xama)
	if xama.Block.PM == nil || *xama.Block.PM != 20 {
		t.Errorf("xamã sem os 20 PM do livro: %+v", xama.Block.PM)
	}
}

// Saber o id não basta: a criatura tem de ser DESTA campanha, senão o mestre de
// uma mesa reescreve o vilão de outra.
func TestACreatureFromAnotherCampaignIsOutOfReach(t *testing.T) {
	s := newTestServer(t)
	mestre := seedUser(t, s, "mestre@t20.local")
	minha := seedCampaign(t, s, mestre)
	outra := seedCampaign(t, s, mestre)

	criada := authed(t, s, mestre, http.MethodPost,
		"/campaigns/"+id64(outra)+"/creatures", corpoCriatura("Vilão", blocoBandido))
	var dto creatureDTO
	_ = json.Unmarshal(criada.Body.Bytes(), &dto)

	alvo := "/campaigns/" + id64(minha) + "/creatures/" + id64(dto.ID)
	if got := authed(t, s, mestre, http.MethodPatch, alvo, corpoCriatura("Roubado", blocoBandido)); got.Code != http.StatusNotFound {
		t.Errorf("PATCH cruzado: %d, queria 404", got.Code)
	}
	if got := authed(t, s, mestre, http.MethodDelete, alvo, ""); got.Code != http.StatusNotFound {
		t.Errorf("DELETE cruzado: %d, queria 404", got.Code)
	}
}

func TestACreatureRefusesAnImpossibleBlock(t *testing.T) {
	s := newTestServer(t)
	mestre := seedUser(t, s, "mestre@t20.local")
	campanha := seedCampaign(t, s, mestre)
	rota := "/campaigns/" + id64(campanha) + "/creatures"

	casos := []struct{ nome, corpo string }{
		{"sem nome", corpoCriatura("  ", blocoBandido)},
		{"tipo fora do livro", corpoCriatura("X", `{"nd":1,"tipo":"dragao","size":"medio","hp":10}`)},
		{"tamanho fora do livro", corpoCriatura("X", `{"nd":1,"tipo":"monstro","size":"gigante","hp":10}`)},
		{"sem vida", corpoCriatura("X", `{"nd":1,"tipo":"monstro","size":"medio","hp":0}`)},
	}
	for _, c := range casos {
		if got := authed(t, s, mestre, http.MethodPost, rota, c.corpo); got.Code != http.StatusBadRequest {
			t.Errorf("%s: %d, queria 400 — %s", c.nome, got.Code, got.Body.String())
		}
	}
}
