package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"t20engine/sheet"
)

// AS REGRAS DE ESCOLHA de poder (ALE-272, fatia 8).
//
// # Elas eram só da TELA, e o servidor gravava qualquer coisa
//
// Quantos poderes cabem no nível, quantos benefícios a origem dá, quais
// caminhos e quais deuses cada classe aceita: tudo isso vivia em 363 linhas de
// `shared/rules/abilities-*.ts`, e o `handleUpdateAbilities` gravava os cinco
// blobs sem conferir NADA. Um pedido montado à mão punha vinte poderes num
// personagem de nível 1 — e o motor somava os modificadores de todos.
//
// É a quarta fronteira desta épica, e a maior. Decisão do dono nesta fatia:
// fechar de forma ESTRITA — a escrita de escolhas tem de deixar a ficha VÁLIDA,
// e não só "não piorar". O projeto ainda não foi usado numa mesa real, então não
// há ficha antiga fora da conta para proteger.

// oLimiteDeBeneficiosDaOrigem são os benefícios que a origem concede: duas
// perícias e um poder é o desenho do livro (p85), e a ficha os trata como DOIS
// itens de uma lista que inclui o poder único.
const oLimiteDeBeneficiosDaOrigem = 2

// oPrimeiroNivelComPoder é o nível em que a primeira vaga de poder abre. Todas
// as catorze classes ganham "um poder por nível a partir do 2º" (p33).
const oPrimeiroNivelComPoder = 2

// asVagasDePoder são as vagas que o nível na classe já abriu.
func asVagasDePoder(nivel int64) int {
	if nivel < oPrimeiroNivelComPoder {
		return 0
	}
	return int(nivel) - oPrimeiroNivelComPoder + 1
}

// osCaminhosDaClasse são as escolhas de caminho, e o nível em que elas abrem.
//
// O Arcanista escolhe no 1º (o caminho DEFINE o atributo-chave dele), e o
// Paladino e o Cavaleiro no 5º.
var osCaminhosDaClasse = map[string]struct {
	Opcoes   []filterOption
	MinLevel int64
}{
	"Arcanista": {Opcoes: []filterOption{
		{Valor: "bruxo", Rotulo: "Bruxo"},
		{Valor: "feiticeiro", Rotulo: "Feiticeiro"},
		{Valor: "mago", Rotulo: "Mago"},
	}, MinLevel: 1},
	"Paladino": {Opcoes: []filterOption{
		{Valor: "egide-sagrada", Rotulo: "Égide Sagrada"},
		{Valor: "montaria-sagrada", Rotulo: "Montaria Sagrada"},
	}, MinLevel: 5},
	"Cavaleiro": {Opcoes: []filterOption{
		{Valor: "bastiao", Rotulo: "Bastião"},
		{Valor: "montaria", Rotulo: "Montaria"},
	}, MinLevel: 5},
}

// osDevotosDaClasse são os deuses que a classe aceita, ou nil quando ela não
// escolhe devoto.
//
// As três listas são do livro: o Clérigo serve qualquer deus MAIOR ou o Panteão
// (p57); o Paladino tem a lista de oito mais o "Paladino do Bem" (p82); o
// Druida serve Allihanna, Megalokk ou Oceano (p61), e não tem alternativa fora
// das divindades.
func osDevotosDaClasse(classe string) []filterOption {
	_, _, deuses := catalogosDoPersonagem()
	switch classe {
	case "Clérigo":
		return append(osDeusesQue(deuses, func(d deusDoLivro) bool { return d.Major }),
			filterOption{Valor: "panteao", Rotulo: "Panteão"})
	case "Paladino":
		return append(osDeusesQue(deuses, func(d deusDoLivro) bool { return d.PaladinoEligible }),
			filterOption{Valor: "paladino-do-bem", Rotulo: "Paladino do Bem"})
	case "Druida":
		return osDeusesQue(deuses, func(d deusDoLivro) bool { return d.DruidaEligible })
	}
	return nil
}

func osDeusesQue(deuses []deusDoLivro, aceita func(deusDoLivro) bool) []filterOption {
	fora := []filterOption{}
	for _, d := range deuses {
		if aceita(d) {
			fora = append(fora, filterOption{Valor: d.ID, Rotulo: d.Name})
		}
	}
	return fora
}

// ── A VALIDAÇÃO, que é a fronteira ───────────────────────────────────────────

// aFichaComEscolhasValidas recusa um conjunto de escolhas que o livro não
// permite.
//
// Ela vale sobre o RESULTADO, e não sobre a diferença: a escrita tem de deixar a
// ficha inteira válida. É mais estrito que "não acrescente além do limite" — uma
// ficha que já esteja fora da conta não aceita escrita de escolha nenhuma até
// ser arrumada — e é a decisão do dono nesta fatia.
func aFichaComEscolhasValidas(dto sheet.CharacterDTO) error {
	if err := osPoderesEscolhidosCabem(dto); err != nil {
		return err
	}
	if err := osBeneficiosDeOrigemCabem(dto); err != nil {
		return err
	}
	return asEscolhasDeClasseValem(dto)
}

// osPoderesEscolhidosCabem confere a conta de vagas e a procedência de cada id.
//
// As vagas são a SOMA das classes: um bárbaro 3/ladino 2 tem as vagas dos dois
// níveis. E cada poder escolhido precisa existir — um poder de classe ELETIVO
// de uma classe que o personagem tem, ou um poder geral. Automático não conta:
// ele não ocupa vaga porque não foi escolhido.
func osPoderesEscolhidosCabem(dto sheet.CharacterDTO) error {
	escolhidos := asEscolhasGuardadas(dto.ClassPowers)
	vagas := 0
	classes := map[string]bool{}
	for _, c := range dto.Classes {
		vagas += asVagasDePoder(c.Level)
		classes[c.ClassName] = true
	}
	for _, id := range escolhidos {
		if err := oPoderEscolhidoExiste(id, classes); err != nil {
			return err
		}
	}
	if len(escolhidos) > vagas {
		return fmt.Errorf("são %d poderes escolhidos para %s",
			len(escolhidos), asVagasEscritas(vagas))
	}
	return nil
}

func oPoderEscolhidoExiste(id string, classes map[string]bool) error {
	if poder, tem := poderesDeClasseDoLivro()[id]; tem {
		if poder.GrantedAtLevel != nil {
			return fmt.Errorf("%q é automático da classe e não ocupa vaga", poder.Name)
		}
		if !classes[poder.ClassName] {
			return fmt.Errorf("%q é um poder de %s, e esta ficha não tem a classe",
				poder.Name, poder.ClassName)
		}
		return nil
	}
	if _, tem := poderesGeraisDoLivro()[id]; tem {
		return nil
	}
	return fmt.Errorf("o poder %q não existe no livro", id)
}

func asVagasEscritas(vagas int) string {
	if vagas == 1 {
		return "1 vaga"
	}
	return strconv.Itoa(vagas) + " vagas"
}

// osBeneficiosDeOrigemCabem confere o teto de dois e a procedência.
func osBeneficiosDeOrigemCabem(dto sheet.CharacterDTO) error {
	escolhidos := asEscolhasGuardadas(dto.OriginChoices)
	if len(escolhidos) > oLimiteDeBeneficiosDaOrigem {
		return fmt.Errorf("a origem dá %d benefícios, e foram escolhidos %d",
			oLimiteDeBeneficiosDaOrigem, len(escolhidos))
	}
	origem, tem := origensDoLivro()[dto.Origin]
	if !tem {
		return nil
	}
	daOrigem := map[string]bool{}
	for _, b := range osBeneficiosQueAOrigemOferece(origem) {
		daOrigem[b.ID] = true
	}
	for _, id := range escolhidos {
		if !daOrigem[id] {
			return fmt.Errorf("%q não é um benefício de %s", id, origem.Name)
		}
	}
	return nil
}

// asEscolhasDeClasseValem confere caminho e devoto contra as opções da classe.
func asEscolhasDeClasseValem(dto sheet.CharacterDTO) error {
	escolhas := asEscolhasDeClasse(dto)
	for _, classe := range dto.Classes {
		blob := escolhas[classe.ClassName]
		if err := aOpcaoEscolhidaExiste(
			"caminho", blob.Caminho, osCaminhosDoNivel(classe.ClassName, classe.Level), classe.ClassName,
		); err != nil {
			return err
		}
		if err := aOpcaoEscolhidaExiste(
			"devoto", blob.Devoto, osDevotosDaClasse(classe.ClassName), classe.ClassName,
		); err != nil {
			return err
		}
	}
	return nil
}

// osCaminhosDoNivel são as opções de caminho quando o nível já as abriu.
func osCaminhosDoNivel(classe string, nivel int64) []filterOption {
	slot, tem := osCaminhosDaClasse[classe]
	if !tem || nivel < slot.MinLevel {
		return nil
	}
	return slot.Opcoes
}

// aOpcaoEscolhidaExiste recusa um valor fora da lista da classe.
//
// Vazio é caminho normal: quem ainda não escolheu tem uma PENDÊNCIA, e não um
// erro — a ficha existe para ser preenchida aos poucos.
func aOpcaoEscolhidaExiste(qual, valor string, opcoes []filterOption, classe string) error {
	if valor == "" {
		return nil
	}
	if len(opcoes) == 0 {
		return fmt.Errorf("%s não escolhe %s", classe, qual)
	}
	for _, o := range opcoes {
		if o.Valor == valor {
			return nil
		}
	}
	return fmt.Errorf("%q não é um %s de %s", valor, qual, classe)
}

// asEscolhasGuardadas lê um blob de ids. Blob torto vira lista vazia — a mesma
// degradação das proficiências, pelo mesmo motivo: a ficha não pode deixar de
// abrir porque uma linha do banco está errada.
func asEscolhasGuardadas(blob string) []string {
	var ids []string
	if json.Unmarshal([]byte(blob), &ids) != nil {
		return nil
	}
	return ids
}

// ── AS PENDÊNCIAS ────────────────────────────────────────────────────────────

// pendencia é uma escolha que ainda falta fazer.
type pendencia struct {
	// Fonte é `raca`, `origem` ou `classe` — a aba do diálogo que a resolve.
	Fonte  string
	Rotulo string
}

// asPendenciasDaFicha são as escolhas que faltam, na ordem das abas.
func (s *Server) asPendenciasDaFicha(dto sheet.CharacterDTO) []pendencia {
	fora := []pendencia{}
	fora = append(fora, s.aPendenciaDoAtributoDeRaca(dto)...)
	fora = append(fora, aPendenciaDaOrigem(dto)...)
	fora = append(fora, asPendenciasDeClasse(dto)...)
	return fora
}

// aPendenciaDoAtributoDeRaca é o `+1 ×3` do humano e a ascendência do suraggel.
//
// Ela PERGUNTA ao motor em vez de repetir a condição dele: o `resolveAtributoMod`
// já sabe quantas escolhas cada raça pede, que elas têm de ser distintas e qual
// atributo é proibido. Repetir as três regras aqui seria a asserção que se
// re-deriva da implementação, com a garantia de divergir no dia em que uma raça
// nova tiver uma quarta condição.
func (s *Server) aPendenciaDoAtributoDeRaca(dto sheet.CharacterDTO) []pendencia {
	if s.catalogs == nil {
		return nil
	}
	fora := []pendencia{}
	for _, r := range dto.Races {
		if s.catalogs.RaceAttributeChoiceIsComplete(r.Race, dto.RaceAttributeChoices) {
			continue
		}
		fora = append(fora, pendencia{
			Fonte: "raca", Rotulo: "Raça: distribuir o bônus de atributo de " + r.Race,
		})
	}
	return fora
}

func aPendenciaDaOrigem(dto sheet.CharacterDTO) []pendencia {
	origem, tem := origensDoLivro()[dto.Origin]
	if !tem {
		return nil
	}
	// A COBRANÇA É PELO QUE A ORIGEM OFERECE, e não pelo teto de dois. O
	// Amnésico é a exceção que ensinou: ele tem ZERO benefícios na lista, porque
	// "em vez de dois benefícios, recebe uma perícia e um poder escolhidos pelo
	// mestre" (p88) — cobrar dois dele daria uma pendência que a pessoa não tem
	// como resolver, para sempre.
	oferece := len(osBeneficiosQueAOrigemOferece(origem))
	teto := oLimiteDeBeneficiosDaOrigem
	if oferece < teto {
		teto = oferece
	}
	faltam := teto - len(asEscolhasGuardadas(dto.OriginChoices))
	if faltam <= 0 {
		return nil
	}
	palavra := "benefícios"
	if faltam == 1 {
		palavra = "benefício"
	}
	return []pendencia{{
		Fonte: "origem", Rotulo: "Origem: " + strconv.Itoa(faltam) + " " + palavra + " por escolher",
	}}
}

func asPendenciasDeClasse(dto sheet.CharacterDTO) []pendencia {
	escolhas := asEscolhasDeClasse(dto)
	usadas := len(asEscolhasGuardadas(dto.ClassPowers))
	fora := []pendencia{}
	for _, classe := range dto.Classes {
		if faltam := asVagasDePoder(classe.Level) - usadas; faltam > 0 {
			palavra := "poderes"
			if faltam == 1 {
				palavra = "poder"
			}
			fora = append(fora, pendencia{
				Fonte:  "classe",
				Rotulo: classe.ClassName + ": " + strconv.Itoa(faltam) + " " + palavra + " por escolher",
			})
		}
		blob := escolhas[classe.ClassName]
		if len(osDevotosDaClasse(classe.ClassName)) > 0 && blob.Devoto == "" {
			fora = append(fora, pendencia{
				Fonte: "classe", Rotulo: classe.ClassName + ": escolher devoto",
			})
		}
		if len(osCaminhosDoNivel(classe.ClassName, classe.Level)) > 0 && blob.Caminho == "" {
			fora = append(fora, pendencia{
				Fonte: "classe", Rotulo: classe.ClassName + ": escolher caminho",
			})
		}
	}
	return fora
}

// asPendenciasEscritas é "3 escolhas pendentes", com o singular certo.
func asPendenciasEscritas(total int) string {
	if total == 1 {
		return "1 escolha pendente"
	}
	return strconv.Itoa(total) + " escolhas pendentes"
}

// aFonteEscrita é o rótulo da aba do diálogo.
var asFontesEscritas = map[string]string{"raca": "Raça", "origem": "Origem", "classe": "Classe"}

func aFonteEscrita(fonte string) string {
	if nome, tem := asFontesEscritas[fonte]; tem {
		return nome
	}
	return strings.ToUpper(fonte)
}
