package api

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"t20engine/app"
	"t20engine/app/campaign"
	"t20engine/domain/book"
	"t20engine/domain/creature"
	"t20engine/serve/web/master"
)

// TODO GESTO DO ELENCO PASSA PELA TRAVA DE CAMPANHA.
//
// # O que a trava protege
//
// O id do NPC vem do CAMINHO ou do rascunho, e os dois são digitáveis. Sem ela,
// o mestre de uma mesa alcança o elenco de OUTRA — e o elenco guarda a
// preparação da campanha, que é o material mais privado que um mestre tem.
//
// Ela existia em dois lugares antes da ALE-353: na cena da Mesa e no
// `app/initiative`, com um comentário na cena avisando *"Duas cópias dariam duas
// travas, e a que envelhecesse seria a de menos uso"*. Hoje é uma, no
// `campaign.Cast`.
//
// # Por que REFLEXÃO e não uma lista de quatro casos
//
// Porque a lista escrita à mão é o que envelhece. Um gesto NOVO no `Cast` — e
// vão nascer, porque o elenco ainda vai ganhar importar e exportar — entraria
// sem trava e sem nada acusar, que é o modo de falha desta família inteira.
//
// O critério é mecânico e não uma enumeração: **todo método exportado que
// recebe um `app.Caller` é um gesto autorizado**, e o caso o exercita com alguém
// que não é dono da campanha. Um método que não receba `app.Caller` aparece no
// relatório do denominador, para a decisão de deixá-lo de fora ser DITA e não
// esquecida — é o caso do `List`, que é leitura e cuja porta é gateada pela
// rota que a desenha.
func TestEveryCastGestureGoesThroughTheCampaignLock(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t.com")
	intruso := seedUser(t, s, "intruso@t.com")
	alheia := seedCampaign(t, s, dono)
	elenco := s.campaignCast()

	// Um NPC REAL na campanha alheia: sem ele, um gesto poderia recusar por
	// "não existe" e o caso leria isso como se fosse a trava.
	npcID, err := elenco.Save(context.Background(), app.Caller{ID: dono}, alheia,
		"Ogro do dono", aBookBlock(t))
	if err != nil {
		t.Fatalf("semear o npc da campanha alheia: %v", err)
	}

	gestos := map[string]func() error{
		"Save": func() error {
			_, e := elenco.Save(context.Background(), app.Caller{ID: intruso}, alheia,
				"Intruso", aBookBlock(t))
			return e
		},
		"Update": func() error {
			return elenco.Update(context.Background(), app.Caller{ID: intruso}, alheia, npcID,
				"Renomeado pelo intruso", aBookBlock(t))
		},
		"Erase": func() error {
			_, e := elenco.Erase(context.Background(), app.Caller{ID: intruso}, alheia, npcID)
			return e
		},
		"Block": func() error {
			_, _, e := elenco.Block(context.Background(), app.Caller{ID: intruso}, alheia, npcID)
			return e
		},
		"CloneBlock": func() error {
			_, e := elenco.CloneBlock(context.Background(), app.Caller{ID: intruso}, alheia, npcID, "Cópia")
			return e
		},
	}

	autorizados, semCaller := gesturesTakingACaller(t)
	for _, nome := range autorizados {
		exercita, temCaso := gestos[nome]
		if !temCaso {
			t.Errorf("o gesto %q do `campaign.Cast` recebe um `app.Caller` e este caso não o "+
				"exercita.\nTodo gesto autorizado tem de ser recusado para quem não é dono "+
				"da campanha — acrescente a linha no mapa acima.", nome)
			continue
		}
		if err := exercita(); !errors.Is(err, app.ErrForbidden) {
			t.Errorf("o gesto %q aceitou um intruso: devolveu %v, e a trava devolve "+
				"`app.ErrForbidden`", nome, err)
		}
	}

	// O DENOMINADOR, e ele tem duas metades. A primeira: um `Cast` que perdesse
	// os métodos daria zero gestos e o laço acima passaria sobre nada.
	if len(autorizados) < 5 {
		t.Fatalf("a reflexão achou só %d gestos autorizados no `campaign.Cast` — "+
			"ou eles sumiram, ou o critério do `app.Caller` deixou de casar", len(autorizados))
	}
	// A segunda: o que ficou de FORA aparece, para a decisão ser dita.
	t.Logf("gestos autorizados: %v | sem `app.Caller` (leitura): %v", autorizados, semCaller)
	if len(semCaller) != 1 || semCaller[0] != "List" {
		t.Errorf("o `campaign.Cast` tem %v sem `app.Caller`, e só o `List` devia estar aí.\n"+
			"Um gesto que ESCREVE sem receber quem pede não tem como ser recusado.", semCaller)
	}
}

// aBookBlock é o bloco do Ogro, copiado do verbete como a cena faz.
//
// Escrever um bloco à mão aqui faria o caso reprovar na VALIDAÇÃO e não na
// trava — foi o que aconteceu na primeira corrida, e a mensagem dizia "o tipo
// não é um dos tipos de criatura do livro".
func aBookBlock(t *testing.T) creature.Block {
	t.Helper()
	v := book.EntryByID("ogro")
	if v == nil {
		t.Fatal("o ogro saiu do bestiário — o caso precisa de um verbete de verdade")
	}
	return master.CopyOfEntry(*v)
}

// gesturesTakingACaller separa os métodos exportados do `Cast` pelos que recebem
// `app.Caller` — que é a marca de "este gesto é autorizado".
func gesturesTakingACaller(t *testing.T) (comCaller, semCaller []string) {
	t.Helper()
	oCaller := reflect.TypeOf(app.Caller{})
	tipo := reflect.TypeOf(campaign.Cast{})
	for i := 0; i < tipo.NumMethod(); i++ {
		m := tipo.Method(i)
		pede := false
		for a := 1; a < m.Type.NumIn(); a++ {
			if m.Type.In(a) == oCaller {
				pede = true
				break
			}
		}
		if pede {
			comCaller = append(comCaller, m.Name)
		} else {
			semCaller = append(semCaller, m.Name)
		}
	}
	return comCaller, semCaller
}
