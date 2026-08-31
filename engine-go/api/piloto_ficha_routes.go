package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/catalog"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/plataforma"
)

// As rotas da FICHA no piloto (ALE-272, fatia 1).
//
// A ficha nova mora em `/piloto/personagens/{id}` — filha do endereço do elenco,
// e não numa raiz própria: ela é o que se abre DE dentro da lista, e o endereço
// dizer isso é o que faz o ‹ Voltar ter para onde voltar.
//
// NENHUM link aponta para cá ainda, e é deliberado: enquanto os painéis não
// existirem, mandar quem clica num herói para uma casca vazia seria uma
// regressão. A virada dos `href` é a última fatia desta issue, como foi na
// ALE-269 — até lá esta cena se alcança por URL, que é o que a bancada e os
// guardas usam.

func (s *Server) rotasDaFicha(r chi.Router) {
	r.Get("/personagens/{id}", s.handleFicha)
	// O PASSO no caminho e não no corpo, como o quadrado do movimento no
	// tabuleiro: o valor é do botão que foi clicado, e não de um sinal da página
	// que quatro botões disputariam.
	r.Post("/personagens/{id}/vitais/{qual}/{passo}", s.comandoDaFicha(mexeNoVital))
	// A CLASSE vai no caminho porque o nível é dela: o do personagem é a SOMA.
	r.Post("/personagens/{id}/nivel/{classe}/{passo}", s.comandoDaFicha(mudaONivel))
	// DOIS caminhos e não um com "padrao" no lugar da categoria: aí o dia em que
	// uma proficiência se chamasse `padrao` calaria o restaurar, e o `alterna`
	// no meio é o que impede a colisão de existir.
	r.Post("/personagens/{id}/proficiencias/alterna/{categoria}", s.comandoDaFicha(alternaAProficiencia))
	r.Post("/personagens/{id}/proficiencias/padrao", s.comandoDaFicha(restauraOPadraoDaClasse))
	// AS PERÍCIAS (fatia 4). O NOME vai no CAMINHO, escapado, e não num sinal:
	// é a mesma decisão do `oComandoDoDegrau` com a classe, e o `handleDelete`
	// da API JSON já endereça a perícia assim. Um sinal aqui seria disputado por
	// 29 botões idênticos.
	r.Post("/personagens/{id}/pericias/treino/{nome}", s.comandoDaFicha(alternaOTreino))
	r.Post("/personagens/{id}/pericias/atributo/{nome}/{atributo}", s.comandoDaFicha(trocaOAtributo))
	r.Post("/personagens/{id}/pericias/remove/{nome}", s.comandoDaFicha(removeOOficio))
	// A CRIAÇÃO é a única que lê SINAL, porque o nome é texto que a pessoa acabou
	// de digitar e ainda não existe em lugar nenhum para virar caminho.
	r.Post("/personagens/{id}/pericias/nova", s.comandoDaFicha(criaOOficio))
	// OS EFEITOS (fatia 5). Quatro donos de estado, quatro caminhos.
	r.Post("/personagens/{id}/efeitos/condicao/{cond}", s.comandoDaFicha(toggleBookCondition))
	r.Post("/personagens/{id}/efeitos/aplica/{magia}", s.comandoDaFicha(applySpellBuff))
	r.Post("/personagens/{id}/efeitos/encerra/{efeito}", s.comandoDaFicha(endAppliedEffect))
	r.Post("/personagens/{id}/efeitos/postura/{flag}", s.comandoDaFicha(endStance))
	// O SITUACIONAL manda a CHAVE do condicional, que é um encadeado com `::` e
	// texto livre do catálogo dentro — por isso ela vai por SINAL e não no
	// caminho. É a exceção da ficha, e a razão dela é o formato da chave.
	r.Post("/personagens/{id}/efeitos/situacao", s.comandoDaFicha(toggleSituational))
	// AS MAGIAS (fatia 6).
	r.Post("/personagens/{id}/magias/aprende/{magia}", s.comandoDaFicha(learnSpell))
	r.Post("/personagens/{id}/magias/esquece/{magia}", s.comandoDaFicha(forgetSpell))
	r.Post("/personagens/{id}/magias/prepara/{magia}", s.comandoDaFicha(togglePrepared))
	r.Post("/personagens/{id}/magias/conjura/{magia}", s.comandoDaFicha(castSpellFromSheet))
	// A MOCHILA (fatia 7).
	r.Post("/personagens/{id}/itens/{item}/guarda", s.comandoDaFicha(stowItem))
	r.Post("/personagens/{id}/itens/{item}/equipa/{slot}", s.comandoDaFicha(equipItemFromSheet))
	// O DINHEIRO manda o modo e o valor por SINAL: a conta do saldo depende dos
	// dois juntos, e um caminho com o valor dentro daria um endereço diferente
	// a cada tecla digitada no campo.
	r.Post("/personagens/{id}/dinheiro", s.comandoDaFicha(changeMoney))
	r.Post("/personagens/{id}/itens/adiciona/{catalogo}", s.comandoDaFicha(addCatalogItem))
	r.Post("/personagens/{id}/itens/custom", s.comandoDaFicha(addCustomItem))
	r.Post("/personagens/{id}/itens/{item}/edita", s.comandoDaFicha(editItem))
	r.Post("/personagens/{id}/itens/{item}/remove", s.comandoDaFicha(removeItemFromSheet))
	r.Post("/personagens/{id}/itens/{item}/usa", s.comandoDaFicha(useItem))
	r.Post("/personagens/{id}/itens/{item}/melhorias", s.comandoDaFicha(applyOverlays))
}

// addCatalogItem põe na mochila um item do Capítulo 3.
//
// O NOME e os ESPAÇOS vêm do catálogo, e não do cliente: são dado transcrito do
// livro, e deixar o navegador mandá-los abriria a porta para uma "Espada longa"
// de 0 espaços.
func addCatalogItem(s *Server, r *http.Request, row sqlcgen.Character, sinais fichaSignals) error {
	catalogo := itemDoLivroPorID(chi.URLParam(r, "catalogo"))
	if catalogo == nil {
		return fmt.Errorf("o item %q não existe no livro", chi.URLParam(r, "catalogo"))
	}
	quantidade, err := aQuantidadePedida(sinais)
	if err != nil {
		return err
	}
	_, err = s.queries.CreateItem(r.Context(), sqlcgen.CreateItemParams{
		Characterid: row.ID, Catalogid: sql.NullString{String: catalogo.ID, Valid: true},
		Name: catalogo.Name, Quantity: quantidade, Slots: catalogo.Slots,
		Improvements: "[]", Createdat: plataforma.NowISO(),
	})
	return err
}

// addCustomItem cria o item que o livro não tem — a lembrança de um NPC, a
// chave de um cofre.
func addCustomItem(s *Server, r *http.Request, row sqlcgen.Character, sinais fichaSignals) error {
	nome, quantidade, espacos, err := oItemCustomPedido(sinais)
	if err != nil {
		return err
	}
	_, err = s.queries.CreateItem(r.Context(), sqlcgen.CreateItemParams{
		Characterid: row.ID, Name: nome, Quantity: quantidade, Slots: espacos,
		Improvements: "[]", Createdat: plataforma.NowISO(),
	})
	return err
}

// editItem muda nome, quantidade e espaços de um item já na ficha.
func editItem(s *Server, r *http.Request, row sqlcgen.Character, sinais fichaSignals) error {
	item, err := s.oItemDaFicha(r, row.ID)
	if err != nil {
		return err
	}
	nome, quantidade, espacos, err := oItemCustomPedido(sinais)
	if err != nil {
		return err
	}
	var set setBuilder
	set.Add("name = ?", nome)
	set.Add("quantity = ?", quantidade)
	set.Add("slots = ?", espacos)
	return set.exec(r.Context(), s.db, "UPDATE character_items", item.ID)
}

// removeItemFromSheet tira o item da ficha.
func removeItemFromSheet(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	item, err := s.oItemDaFicha(r, row.ID)
	if err != nil {
		return err
	}
	return s.queries.DeleteItem(r.Context(), item.ID)
}

// useItem gasta uma dose do consumível.
//
// A regra inteira — a rolagem presa no máximo, a linha de efeito de cena ou dia,
// a porção diária, a baixa do item — é a MESMA da API JSON
// (`consumeItemForCharacter`), extraída nesta fatia. Os números rolados vêm por
// sinal porque quem rola é a MESA: a ficha não rola dado por ninguém.
func useItem(s *Server, r *http.Request, row sqlcgen.Character, sinais fichaSignals) error {
	item, err := s.oItemDaFicha(r, row.ID)
	if err != nil {
		return err
	}
	_, err = s.consumeItemForCharacter(r, row, item.ID, sinais.ItemRolagemPv, sinais.ItemRolagemPm)
	return err
}

// applyOverlays grava as melhorias e o material escolhidos.
//
// A COMPATIBILIDADE é conferida aqui (`aMelhoriaCabeNoItem`), e essa checagem
// não existia em servidor nenhum até a fatia 7: a regra vivia no filtro do
// diálogo da SPA, que some junto com ela.
func applyOverlays(s *Server, r *http.Request, row sqlcgen.Character, sinais fichaSignals) error {
	item, err := s.oItemDaFicha(r, row.ID)
	if err != nil {
		return err
	}
	catalogo := itemDoLivroPorID(oCatalogoDoItem(item))
	if err := aMelhoriaCabeNoItem(catalogo, sinais.ItemMelhorias, "improvement"); err != nil {
		return err
	}
	materiais := []string{}
	if sinais.ItemMaterial != "" {
		materiais = append(materiais, sinais.ItemMaterial)
	}
	if err := aMelhoriaCabeNoItem(catalogo, materiais, "material"); err != nil {
		return err
	}
	var set setBuilder
	set.Add("improvements = ?", marshalStrings(&sinais.ItemMelhorias))
	set.Add("material = ?", sql.NullString{String: sinais.ItemMaterial, Valid: sinais.ItemMaterial != ""})
	return set.exec(r.Context(), s.db, "UPDATE character_items", item.ID)
}

// aQuantidadePedida lê a quantidade, com as mesmas bordas da API JSON.
func aQuantidadePedida(sinais fichaSignals) (int64, error) {
	if sinais.ItemQtd == nil {
		return 1, nil
	}
	if *sinais.ItemQtd < 1 || *sinais.ItemQtd > 9999 {
		return 0, fmt.Errorf("a quantidade %d está fora de 1 a 9999", *sinais.ItemQtd)
	}
	return *sinais.ItemQtd, nil
}

// oItemCustomPedido lê nome, quantidade e espaços, com as bordas do formulário.
//
// Os ESPAÇOS são múltiplos de meio porque é assim que o livro conta carga
// (p141) — e essa é a mesma borda que a API JSON cobra, `slotsNotMultiple`.
func oItemCustomPedido(sinais fichaSignals) (string, int64, float64, error) {
	nome := ""
	if sinais.ItemNome != nil {
		nome = strings.TrimSpace(*sinais.ItemNome)
	}
	if nome == "" {
		return "", 0, 0, fmt.Errorf("informe um nome para o item")
	}
	if len([]rune(nome)) > 80 {
		return "", 0, 0, fmt.Errorf("o nome tem %d letras, e o máximo são 80", len([]rune(nome)))
	}
	quantidade, err := aQuantidadePedida(sinais)
	if err != nil {
		return "", 0, 0, err
	}
	espacos := 1.0
	if sinais.ItemEspacos != nil {
		espacos = *sinais.ItemEspacos
	}
	if espacos < 0 || slotsNotMultiple(espacos) {
		return "", 0, 0, fmt.Errorf("os espaços (%v) têm de ser múltiplos de 0,5", espacos)
	}
	return nome, quantidade, espacos, nil
}

// stowItem tira o item da mão ou do corpo e o devolve à mochila.
//
// Guardar nunca esbarra em teto — ele só LIBERA espaço —, então este comando
// não passa pelas checagens de eixo e de limite que o `equipItemFromSheet` faz.
func stowItem(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	item, err := s.oItemDaFicha(r, row.ID)
	if err != nil {
		return err
	}
	return gravaOEquipado(r, s, item.ID, sql.NullString{})
}

// equipItemFromSheet põe o item na mão ou no corpo.
//
// As DUAS recusas são as mesmas da API JSON, e pela razão de sempre: o eixo do
// item (`equipAxisError` — um escudo não se veste) e os tetos de 2 mãos e 4
// vestidos (`equipLimitCheck`, p141). Escrevê-las de novo aqui daria duas
// regras para a mesma pergunta.
func equipItemFromSheet(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	item, err := s.oItemDaFicha(r, row.ID)
	if err != nil {
		return err
	}
	slot := chi.URLParam(r, "slot")
	if !ehSlotDeEquipar(slot) {
		return fmt.Errorf("%q não é um lugar de equipar", slot)
	}
	// O EIXO sai do catálogo EMBUTIDO, e não do `s.catalogs` que a API JSON usa.
	//
	// Os dois trazem o mesmo `items.json`, mas o `s.catalogs` é primado de um
	// arquivo por caminho de configuração e o próprio `primeCatalogs` diz o que
	// acontece quando ele falta: "mutation validators disabled". Uma regra que
	// se DESLIGA sozinha quando um arquivo some não é uma regra — e a bancada
	// mostrou o preço, com um escudo sendo vestido num teste porque o catálogo
	// do fixture está vazio. O `catalog.Resource` é `go:embed`: ele existe
	// sempre que o binário existe.
	if _, recusa := equipAxisError(oItemComoDoMotor(itemDoLivroPorID(oCatalogoDoItem(item))), slot); recusa != "" {
		return fmt.Errorf("%s", recusa)
	}
	recusa, err := s.equipLimitCheck(r, row.ID, item.ID, slot)
	if err != nil {
		return err
	}
	if recusa != "" {
		return fmt.Errorf("%s", recusa)
	}
	return gravaOEquipado(r, s, item.ID, sql.NullString{String: slot, Valid: true})
}

// gravaOEquipado escreve a coluna, pelo mesmo construtor de UPDATE parcial que
// a API JSON usa — `character_items` não tem `updatedAt`, e é por isso que a
// gravação é `exec` e não `execTouched`.
func gravaOEquipado(r *http.Request, s *Server, itemID int64, valor sql.NullString) error {
	var set setBuilder
	set.Add("equipped = ?", valor)
	return set.exec(r.Context(), s.db, "UPDATE character_items", itemID)
}

// ehSlotDeEquipar aceita só os três lugares do livro.
func ehSlotDeEquipar(slot string) bool {
	return slot == "vested" || slot == "wielded" || slot == "wielded2"
}

// oItemDaFicha lê o item do caminho e CONFERE que ele é desta ficha.
//
// A posse do personagem já foi conferida pelo `comandoDaFicha`; o que falta é a
// do item, e sem ela um id de outra ficha passaria — a consulta é por id e o
// `characterId` só entraria no `UPDATE`, que não acusaria nada por afetar zero
// linhas.
func (s *Server) oItemDaFicha(r *http.Request, characterID int64) (sqlcgen.GetItemRow, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "item"), 10, 64)
	if err != nil {
		return sqlcgen.GetItemRow{}, fmt.Errorf("o item %q não é um número", chi.URLParam(r, "item"))
	}
	item, err := s.queries.GetItem(r.Context(), id)
	if err != nil || item.Characterid != characterID {
		return sqlcgen.GetItemRow{}, fmt.Errorf("o item %d não é desta ficha", id)
	}
	return item, nil
}

// changeMoney recebe, gasta ou corrige o dinheiro.
//
// Os TRÊS modos existem porque são três gestos diferentes na mesa: "achamos 350
// no baú", "paguei 80 pela estalagem", e escrever o total — que é o gesto da
// forja (Tabela 3-1, p140) e o de consertar um erro de digitação (ALE-224).
func changeMoney(s *Server, r *http.Request, row sqlcgen.Character, sinais fichaSignals) error {
	if sinais.TibarValor == nil {
		return fmt.Errorf("informe um valor a partir de 0")
	}
	saldo, erro := oSaldoDepoisDoGesto(row.Tibar, sinais.TibarModo, *sinais.TibarValor)
	if erro != "" {
		return fmt.Errorf("%s", erro)
	}
	return s.queries.SetCharacterTibar(r.Context(), sqlcgen.SetCharacterTibarParams{
		Tibar: saldo, UpdatedAt: plataforma.NowISO(), ID: row.ID,
	})
}

// learnSpell põe uma magia do catálogo no grimório.
func learnSpell(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	id := chi.URLParam(r, "magia")
	if _, conhecida := catalog.LookupSpell(id); !conhecida {
		return fmt.Errorf("a magia %q não existe no livro", id)
	}
	_, err := s.queries.CreateSpell(r.Context(), sqlcgen.CreateSpellParams{
		Characterid: row.ID, Catalogspellid: id, Prepared: 0, Learnedat: plataforma.NowISO(),
	})
	return err
}

// forgetSpell tira a magia do grimório.
func forgetSpell(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	_, err := s.queries.DeleteSpell(r.Context(), sqlcgen.DeleteSpellParams{
		Characterid: row.ID, Catalogspellid: chi.URLParam(r, "magia"),
	})
	return err
}

// togglePrepared prepara ou despreparar uma magia.
//
// O comando manda a MAGIA e não o estado, pela razão de sempre: mandar
// "preparada" perde para o clique repetido e para a segunda aba aberta.
func togglePrepared(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	id := chi.URLParam(r, "magia")
	todas, err := s.queries.ListSpellsByCharacter(r.Context(), row.ID)
	if err != nil {
		return err
	}
	for _, m := range todas {
		if m.Catalogspellid != id {
			continue
		}
		depois := int64(0)
		if m.Prepared == 0 {
			depois = 1
		}
		_, err := s.queries.SetSpellPreparedByCatalog(r.Context(), sqlcgen.SetSpellPreparedByCatalogParams{
			Prepared: depois, CharacterId: row.ID, CatalogSpellId: id,
		})
		return err
	}
	return fmt.Errorf("a magia %q não está no grimório", id)
}

// castSpellFromSheet conjura, cobrando o PM.
//
// A conta e as recusas são as MESMAS da API JSON — preparação, aprimoramentos, o
// teto da p224 com a ressalva do custo mínimo, e o PM disponível. Escrevê-las de
// novo aqui daria duas regras que divergem no dia em que uma mudar, e é
// exatamente o defeito que a ALE-110 registrou: a redução de custo era exibida
// num lugar e ignorada na hora de cobrar.
func castSpellFromSheet(s *Server, r *http.Request, row sqlcgen.Character, sinais fichaSignals) error {
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	return s.castSpellForCharacter(r, dto, chi.URLParam(r, "magia"), sinais.osAprimoramentos())
}

// toggleBookCondition liga ou desliga UMA condição do livro (p394-395).
//
// # Ela avisa a MESA, e isso não é enfeite
//
// O motor deriva Defesa e perícias da condição (ALE-28), então uma condição
// aplicada sem aviso faz o jogador e o mestre verem números DIFERENTES do mesmo
// personagem, sem nada na tela dizendo que discordam. Foi o defeito da ALE-245,
// e o `handleUpdateConditions` da API JSON é o único lugar que o conserta — a
// ficha em Datastar tinha de conquistar o mesmo, senão o porte REGREDIRIA.
//
// O aviso sai DEPOIS da escrita, nunca antes: avisar sobre algo que ainda pode
// falhar faria a mesa buscar o estado velho e acreditar nele.
func toggleBookCondition(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	cond := chi.URLParam(r, "cond")
	if !catalog.IsCondition(cond) {
		return fmt.Errorf("%q não é uma condição do livro", cond)
	}
	atuais := parseConditionBlob(row.Activeconditions)
	depois := []string{}
	tinha := false
	for _, c := range atuais {
		if c == cond {
			tinha = true
			continue
		}
		depois = append(depois, c)
	}
	if !tinha {
		depois = append(depois, cond)
	}
	blob := marshalStrings(&depois)
	if err := s.queries.UpdateConditions(r.Context(), sqlcgen.UpdateConditionsParams{
		ActiveConditions: blob, UpdatedAt: plataforma.NowISO(), ID: row.ID,
	}); err != nil {
		return err
	}
	s.characterChanged(row.ID)
	return nil
}

// applySpellBuff aplica uma magia de bônus como efeito de cena ou dia.
//
// A gravação é a MESMA da API JSON (`applySpellBuffEffect`): duas escritas
// divergiriam no dia em que uma regra nova chegasse, e o escopo padrão de cada
// magia vive no catálogo, não aqui.
func applySpellBuff(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	magia := chi.URLParam(r, "magia")
	if _, _, err := s.applySpellBuffEffect(r.Context(), row.ID, magia, nil); err != nil {
		return err
	}
	return nil
}

// endAppliedEffect encerra um efeito em curso.
func endAppliedEffect(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	id, err := strconv.ParseInt(chi.URLParam(r, "efeito"), 10, 64)
	if err != nil {
		return fmt.Errorf("o efeito %q não é um número", chi.URLParam(r, "efeito"))
	}
	// A POSSE É CONFERIDA ANTES, e a query não a confere por nós: o
	// `DeleteEffectByID` apaga por id e mais nada, então sem esta leitura um
	// pedido montado à mão encerraria o efeito de OUTRO personagem. É a mesma
	// checagem que o `handleDeleteEffect` da API JSON faz.
	meta, err := s.queries.GetActiveEffectMeta(r.Context(), id)
	if err != nil || meta.Characterid != row.ID {
		return fmt.Errorf("o efeito %d não é desta ficha", id)
	}
	return s.queries.DeleteEffectByID(r.Context(), id)
}

// endStance encerra uma postura.
//
// Encerrar apaga a linha da postura E desliga a flag: são duas escritas para uma
// coisa só, e deixar a flag ligada manteria os modificadores em pé numa postura
// que a tela diz encerrada. Entrar continua sendo dos Poderes, onde o PM é
// cobrado — aqui não há como pagar nada.
func endStance(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	flag := chi.URLParam(r, "flag")
	if err := s.queries.RemoveCharacterStance(r.Context(), sqlcgen.RemoveCharacterStanceParams{
		Characterid: row.ID, Flag: flag,
	}); err != nil {
		return err
	}
	return s.removeConditionalsWithFlag(r, row, flag)
}

// toggleSituational liga ou desliga um condicional de contexto.
func toggleSituational(s *Server, r *http.Request, row sqlcgen.Character, sinais fichaSignals) error {
	chave := ""
	if sinais.Situacao != nil {
		chave = *sinais.Situacao
	}
	if chave == "" {
		return fmt.Errorf("o gesto não disse qual efeito situacional alternar")
	}
	atuais, err := s.queries.ListCharacterConditionals(r.Context(), row.ID)
	if err != nil {
		return err
	}
	for _, c := range atuais {
		if c == chave {
			return s.queries.RemoveCharacterConditional(r.Context(), sqlcgen.RemoveCharacterConditionalParams{
				Characterid: row.ID, Conditionalid: chave,
			})
		}
	}
	return s.queries.AddCharacterConditional(r.Context(), sqlcgen.AddCharacterConditionalParams{
		Characterid: row.ID, Conditionalid: chave,
	})
}

// removeConditionalsWithFlag desliga todo condicional que a postura acendia.
func (s *Server) removeConditionalsWithFlag(r *http.Request, row sqlcgen.Character, flag string) error {
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	ec, err := engineCharacterFrom(dto)
	if err != nil || s.catalogs == nil {
		return nil
	}
	for _, c := range engine.ComputeItemEffects(s.catalogs.ActiveItemsFor(ec)).Conditional {
		if c.Flag != flag {
			continue
		}
		_ = s.queries.RemoveCharacterConditional(r.Context(), sqlcgen.RemoveCharacterConditionalParams{
			Characterid: row.ID, Conditionalid: engine.ConditionalID(c),
		})
	}
	return nil
}

// oNomeDaPericia lê o nome do caminho, desescapando como a API JSON faz.
func oNomeDaPericia(r *http.Request) string {
	nome := chi.URLParam(r, "nome")
	if decodificado, err := url.PathUnescape(nome); err == nil {
		return decodificado
	}
	return nome
}

// alternaOTreino liga ou desliga o treino de UMA perícia.
//
// O comando manda a PERÍCIA e não o estado desejado, pela mesma razão da
// proficiência: mandar "treinada" perde para o clique repetido e para a segunda
// aba aberta no mesmo personagem. Quem clica quer INVERTER o que está na tela, e
// o servidor sabe o que está na tela melhor que o botão.
func alternaOTreino(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	nome := oNomeDaPericia(r)
	// O estado ATUAL vem da lista e não do `GetExpertiseMeta`, que devolve só o
	// id e o `custom` — inverter exige saber o que está lá.
	todas, err := s.queries.ListExpertisesByCharacter(r.Context(), row.ID)
	if err != nil {
		return err
	}
	for _, e := range todas {
		if e.Name != nome {
			continue
		}
		depois := e.Trained == 0
		_, err := s.queries.UpdateExpertise(r.Context(), sqlcgen.UpdateExpertiseParams{
			Trained: nullBool(&depois), CharacterId: row.ID, Name: nome,
		})
		return err
	}
	return fmt.Errorf("a perícia %q não é desta ficha", nome)
}

// trocaOAtributo repõe a perícia em outro atributo.
//
// O atributo vai no CAMINHO junto do nome: é o valor do `<option>` escolhido, e
// mandá-lo por sinal faria seis opções de 29 linhas disputarem a mesma chave.
func trocaOAtributo(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	atributo := chi.URLParam(r, "atributo")
	if !attributeKeys[atributo] {
		return fmt.Errorf("%q não é um atributo: são %v", atributo, engine.AttributeKeys)
	}
	nome := oNomeDaPericia(r)
	_, err := s.queries.UpdateExpertise(r.Context(), sqlcgen.UpdateExpertiseParams{
		Attribute: nullString(&atributo), CharacterId: row.ID, Name: nome,
	})
	if err != nil {
		return fmt.Errorf("a perícia %q não é desta ficha", nome)
	}
	return nil
}

// removeOOficio apaga uma perícia INVENTADA pelo jogador.
//
// As 29 do livro não se apagam, e a recusa é do servidor e não da tela: a ficha
// nova não desenha a lixeira numa perícia do livro, mas travar só na UI deixaria
// a regra sem fronteira — quem montar o `@post` à mão apagaria a Fortitude.
func removeOOficio(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	nome := oNomeDaPericia(r)
	meta, err := s.queries.GetExpertiseMeta(r.Context(), sqlcgen.GetExpertiseMetaParams{
		Characterid: row.ID, Name: nome,
	})
	if err != nil {
		return fmt.Errorf("a perícia %q não é desta ficha", nome)
	}
	// A COLUNA decide, e não a lista das 29: `custom` é o que o banco guarda
	// sobre esta linha, enquanto a lista é uma opinião do código sobre o nome. As
	// duas concordam hoje; no dia em que uma perícia nova entrar no livro, a
	// coluna continua certa e a lista fica velha.
	if meta.Custom == 0 {
		return fmt.Errorf("%q é uma perícia do livro e não se remove da ficha", nome)
	}
	return s.queries.DeleteExpertiseByID(r.Context(), meta.ID)
}

// criaOOficio acrescenta uma perícia que o livro não tem — o saber de um ferreiro,
// a arte de um marinheiro.
//
// Ela nasce TREINADA, porque inventar um ofício e não tê-lo treinado não é um
// estado que signifique alguma coisa. A validação é a MESMA da API JSON
// (`guardaOOficioNovo`), extraída na fatia 4: duas validações divergiriam no dia
// em que uma regra nova chegasse, e a esquecida aceitaria o que a outra recusa.
func criaOOficio(s *Server, r *http.Request, row sqlcgen.Character, sinais fichaSignals) error {
	nome, atributo := "", "intelligence"
	if sinais.NovaPericia != nil {
		nome = strings.TrimSpace(*sinais.NovaPericia)
	}
	if sinais.NovoAtributo != nil && attributeKeys[*sinais.NovoAtributo] {
		atributo = *sinais.NovoAtributo
	}
	if err := s.guardaOOficioNovo(r.Context(), row.ID, nome); err != nil {
		return err
	}
	_, err := s.queries.CreateExpertise(r.Context(), sqlcgen.CreateExpertiseParams{
		Characterid: row.ID, Name: nome, Attribute: atributo, Trained: 1, Custom: 1,
	})
	return err
}

// alternaAProficiencia liga ou desliga UMA categoria.
//
// O comando não manda o estado desejado, manda a categoria: mandar "ligada"
// perderia para o clique repetido e para a segunda aba aberta no mesmo
// personagem — quem clica quer INVERTER o que está na tela, e o servidor sabe o
// que está na tela melhor do que o botão sabe.
func alternaAProficiencia(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	depois, err := aTrocaDaProficiencia(dto, chi.URLParam(r, "categoria"))
	if err != nil {
		return err
	}
	return s.gravaAsProficienciasDaFicha(r, row.ID, depois)
}

// restauraOPadraoDaClasse joga fora os ajustes manuais.
func restauraOPadraoDaClasse(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	return s.gravaAsProficienciasDaFicha(r, row.ID, oPadraoDaClasse(dto))
}

// gravaAsProficienciasDaFicha usa a MESMA gravação da API JSON.
//
// A lista de desconhecidas vira frase porque quem está do outro lado é um
// navegador mostrando página, e não um cliente lendo `FieldErrorMap`. Ela só
// dispara se o servidor montar uma categoria que ele próprio não conhece — é o
// guarda contra a tela e a validação divergirem, não contra o jogador.
func (s *Server) gravaAsProficienciasDaFicha(r *http.Request, id int64, categorias []string) error {
	_, desconhecidas, err := s.guardaAsProficiencias(r.Context(), id, categorias)
	if len(desconhecidas) > 0 {
		return fmt.Errorf("proficiência fora do catálogo: %s", strings.Join(desconhecidas, "; "))
	}
	return err
}

func (s *Server) handleFicha(w http.ResponseWriter, r *http.Request) {
	id, ok := oPersonagemDaURL(w, r)
	if !ok {
		return
	}
	sinaisDaPagina := osSinaisDaFicha(r)
	view, status, err := s.carregaFicha(
		r.Context(), currentUser(r), id, aAbaPedida(r.URL.Query().Get("tab")),
		sinaisDaPagina.aBusca(), sinaisDaPagina)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	s.escrevePagina(w, r, http.StatusOK, paginaPiloto{
		Titulo: view.Nome + " · Tormenta 20",
		// `cascaNua`: a cena desenha o próprio cabeçalho, com a volta e o nome.
		Forma: cascaNua,
		// `detalhe` é a caixa do Combate cujo diálogo está aberto, e ele mora no
		// <body> porque o <body> nunca é remendado: declarado dentro do painel, o
		// `@post` do PV o redeclararia a cada toque e fecharia o diálogo que o
		// jogador acabou de abrir. Mesma armadilha do `fichaAberta` do bestiário.
		//
		// Minúsculo de propósito: chave de atributo é minusculada pelo HTML, e um
		// `detalheAberto` ligaria um sinal NOVO em vez do que a expressão lê.
		// `oficio` é o diálogo de criar perícia, e `novapericia`/`novoatributo`
		// são os dois campos dele. Tudo MINÚSCULO: chave de atributo é minusculada
		// pelo HTML, e um `data-bind:novaPericia` ligaria um sinal `novapericia`
		// que o servidor lê — mas o `data-bind` teria escrito noutro, e o campo
		// chegaria sempre vazio.
		Sinais: "{detalhe: '', oficio: false, novapericia: '', novoatributo: 'intelligence'," +
			" condicao: false, buff: false, situacao: ''," +
			" aprender: false, aug0: 0, aug1: 0, aug2: 0, aug3: 0, aug4: 0, aug5: 0," +
			" magiabusca: '', magiacirculo: '', magiaescola: ''," +
			" itembusca: '', itemcategoria: '', tibarmodo: 'receber', tibarvalor: 0," +
			" catalogobusca: '', catalogocategoria: '', itemqtd: 1, itemnome: '', itemespacos: 1," +
			" itemrolagempv: 0, itemrolagempm: 0, itemmelhorias: [], itemmaterial: ''}",
	}, cenaDaFicha(view))
}

// comandoDaFicha é o gateway das mutações da ficha.
//
// Ele existe pela mesma razão do `comandoDoTabuleiro`: resolver a posse, mutar e
// redesenhar são três passos que toda mutação da ficha faz, e escrevê-los em
// cada handler é como um deles esquece de redesenhar — a pessoa clica, o banco
// muda e a tela fica igual.
//
// A POSSE é conferida aqui, uma vez. A ficha é do dono e de mais ninguém: a
// regra é a mesma da API JSON (`characterFor`), e a cena não ganha uma segunda.
func (s *Server) comandoDaFicha(
	mutar func(*Server, *http.Request, sqlcgen.Character, fichaSignals) error,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := oPersonagemDaURL(w, r)
		if !ok {
			return
		}
		// UMA leitura de sinais por requisição, e ela vem ANTES de tudo: o
		// `ReadSignals` consome o corpo do POST, então a segunda chamada
		// receberia vazio sem erro nenhum. Ver `piloto_ficha_sinais.go`.
		sinais := osSinaisDaFicha(r)
		row, err := s.queries.GetCharacter(r.Context(), id)
		if err != nil {
			http.Error(w, "este personagem não existe", http.StatusNotFound)
			return
		}
		if row.Ownerid != currentUser(r).ID {
			http.Error(w, "esta ficha não é sua", http.StatusForbidden)
			return
		}
		// A RECUSA VOLTA PELA CENA, e não por um status de erro.
		//
		// Medido na fatia 7 (ALE-272): o `http.Error(400)` que morava aqui não
		// chegava a lugar nenhum. O cliente do Datastar não aplica remendo de
		// resposta que não é 2xx, então a única marca da recusa era uma linha
		// vermelha no CONSOLE — "Failed to load resource: 400" — e na tela o
		// gesto simplesmente não acontecia. Gastar mais dinheiro do que se tem
		// fechava o diálogo e deixava o saldo igual, sem uma palavra.
		//
		// Todas as recusas da ficha são de REGRA — o teto de duas mãos, o PM que
		// falta, a magia que não está preparada —, e regra recusada é informação
		// para quem joga. Ela sobe com a cena inteira redesenhada, que é o que
		// prova que nada mudou, mais a frase. A API JSON continua respondendo os
		// status dela; esta rota desenha página.
		recusa := ""
		if err := mutar(s, r, row, sinais); err != nil {
			recusa = err.Error()
		}
		view, status, err := s.carregaFicha(
			r.Context(), currentUser(r), id, aAbaPedida(r.URL.Query().Get("tab")), sinais.aBusca(), sinais)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		view.Recusa = recusa
		sse := datastar.NewSSE(w, r)
		fragmento, err := renderFragmento(r.Context(), cenaDaFicha(view))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragmento)
	}
}

// mexeNoVital soma o passo ao PV ou ao PM, PRENDENDO na faixa.
//
// PRENDER e não recusar, e essa é a diferença entre o gesto e a API: o corpo do
// `PATCH /vitals` manda o valor ABSOLUTO e é recusado fora da faixa, o que está
// certo para um cliente que calculou. Aqui o gesto é "levou seis" — com 4 de PV
// o resultado é zero, e uma recusa faria o mestre clicar quatro vezes de um em
// um para chegar no mesmo lugar.
//
// O TETO é o máximo do personagem: curar além do máximo não é PV temporário, que
// é outra regra e tem dono no motor (`TempHpFuria`).
func mexeNoVital(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	passo, err := oPassoDaURL(r)
	if err != nil {
		return err
	}
	qual := chi.URLParam(r, "qual")
	hp, mp := row.Hpcurrent, row.Mpcurrent
	switch qual {
	case "pv":
		hp = presoNaFaixa(hp+int64(passo), row.Hpmax)
	case "pm":
		mp = presoNaFaixa(mp+int64(passo), row.Mpmax)
	default:
		return fmt.Errorf("vital %q não existe: são 'pv' e 'pm'", qual)
	}
	return s.queries.SetVitalsCurrent(r.Context(), sqlcgen.SetVitalsCurrentParams{
		HpCurrent: hp, MpCurrent: mp, UpdatedAt: plataforma.NowISO(), ID: row.ID,
	})
}

// presoNaFaixa mantém o vital entre zero e o máximo.
func presoNaFaixa(valor, max int64) int64 {
	if valor < 0 {
		return 0
	}
	if valor > max {
		return max
	}
	return valor
}

// mudaONivel sobe ou desce UMA CLASSE, e o nível do personagem acompanha.
//
// A primeira versão desta função escrevia direto no nível do personagem, e
// **estava errada** — descobri comparando com a SPA no navegador, não lendo o
// código: lá o degrau chama `PATCH /classes/level`, porque o nível do
// personagem é a SOMA dos níveis de classe. Escrever o total direto deixa a
// ficha dizendo 13 com as classes somando 12, e os pools de PV e PM (que
// derivam das CLASSES) não se mexem — o número sobe e o personagem não fica
// mais forte.
//
// A regra é a MESMA do handler JSON, extraída para os dois usarem (ver
// `aplicaONivelDaClasse`): a classe tem de ser do personagem, o total é limitado
// a 20, e os pools sincronizam.
func mudaONivel(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	passo, err := oPassoDaURL(r)
	if err != nil {
		return err
	}
	classe, err := url.PathUnescape(chi.URLParam(r, "classe"))
	if err != nil {
		return fmt.Errorf("classe %q não é um nome válido", chi.URLParam(r, "classe"))
	}
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	for _, cl := range dto.Classes {
		if cl.ClassName != classe {
			continue
		}
		alvo := cl.Level + int64(passo)
		if alvo < 1 {
			return fmt.Errorf("%s está no nível 1: descer apagaria a classe", classe)
		}
		_, _, _, _, err := s.aplicaONivelDaClasse(r, row, classe, alvo)
		return err
	}
	return fmt.Errorf("%s não é uma classe deste personagem", classe)
}

// oPersonagemDaURL lê o id do caminho. Erro aqui é URL digitada errada, e a
// resposta é uma frase: quem está do outro lado é um navegador mostrando página.
func oPersonagemDaURL(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "o personagem precisa ser um número", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// oPassoDaURL aceita o sinal de menos: o passo é para os dois lados.
func oPassoDaURL(r *http.Request) (int, error) {
	bruto := chi.URLParam(r, "passo")
	passo, err := strconv.Atoi(bruto)
	if err != nil || passo == 0 {
		return 0, fmt.Errorf("passo %q não é um número diferente de zero", bruto)
	}
	return passo, nil
}
