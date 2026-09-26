#!/usr/bin/env python3
"""Confere `gods.json` contra o Capítulo 1 do livro (p96-105). Seção da ALE-391.

DUAS leituras, como nas perícias
---------------------------------
1. **A Tabela 1-20: Deuses** (p97): `Divindade | Energia | Poderes Concedidos`.
2. **O verbete**, que tem os campos ROTULADOS — `Símbolo Sagrado.`,
   `Canalizar Energia.`, `Arma Preferida.`, `Devotos.`, `Poderes Concedidos.`

Elas se conferem uma à outra em energia e poderes ANTES de encostar no
catálogo. O verbete acrescenta símbolo, arma, devotos e a página.

Como o verbete se liga ao deus
-------------------------------
O capítulo se corta em `Crenças e Objetivos.`, que abre todo verbete: 21
ocorrências, sendo a primeira a seção "Características dos deuses", que EXPLICA
os rótulos e não tem `Poderes Concedidos.`.

A ligação bloco→deus é GEOMÉTRICA — o último título de deus antes do corte, em
ordem de leitura por coluna —, e a ordem alfabética é o CONTROLE, não a regra.
Usar o alfabeto como regra funcionaria hoje e calaria no dia em que o livro
mudasse de ordem; usá-lo como controle acusa o dia em que a geometria falhar.

O que NÃO é medível por igualdade
----------------------------------
O `simbolo` e o `armaPreferida` do catálogo são prosa REESCRITA: o livro diz
"Um olho macabro de pupila vertical e cercado de espinhos" e o catálogo tira o
artigo. Comparar frase com frase dá 6 de 20 e o ruído esconde o sinal. O que se
mede é CONTENÇÃO de palavras de conteúdo — o catálogo não pode dizer o que o
livro não diz. Assim o estilo editorial passa e a invenção reprova.

Os `devotos` são vocabulário FECHADO (raça ou classe), então a comparação é de
conjunto sobre termos conhecidos. O que o auditor não reconhece ele **imprime
numa lista nomeada** em vez de descartar em silêncio: a prosa do livro tem
palavras que não são termo ("membros", "podem", "veja"), e o que essa lista
serve para conferir é que nenhuma delas é raça ou classe. É a mesma escolha do
balde de falso positivo classificado do `audit-classes.py`.

Uso: `python3 scripts/audit-gods.py`. Precisa do PDF do livro (veja t20pdf).
"""
import json
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from t20pdf import (  # noqa: E402
    RAIZ, chave, junta, linhas_com_coordenada, normaliza_frase, sem_lixo)

PRIMEIRA, ULTIMA = 102, 111  # PDF
OFFSET_DO_PDF = 6  # livro = PDF - 6
PAGINA_DA_TABELA = 103  # p97, onde a Tabela 1-20 divide a página com o verbete
TABELA_ACABA_EM_Y = 440.0
DADOS = RAIZ / 'engine-go/domain/catalog/data'

ABRE_O_VERBETE = 'Crenças e Objetivos.'
CAMPOS = ('Símbolo Sagrado', 'Canalizar Energia', 'Arma Preferida',
          'Devotos', 'Poderes Concedidos')
FECHA_O_VERBETE = 'Obrigações & Restrições'
TITULO_MAXIMO = 18  # caracteres; nome de deus é curto, prosa não

# Palavras que não distinguem um símbolo de outro. O catálogo tira o artigo por
# estilo, e mantê-las na conta transformaria estilo em divergência.
VAZIAS = {'um', 'uma', 'o', 'a', 'os', 'as', 'de', 'do', 'da', 'dos', 'das',
          'e', 'ou', 'com', 'em', 'no', 'na', 'ao', 'para', 'outros', 'outro'}
SEM_ARMA = 'não há'

# A ÚNICA compressão editorial aceita, nomeada uma a uma — é um PERMITIDOS, e
# uma segunda compressão passa a reprovar em vez de se esconder num total.
# Allihanna: o livro diz "o símbolo corresponde ao respectivo animal" e o
# catálogo resume em "animal totêmico", que é a mesma coisa em duas palavras.
COMPRESSAO_ACEITA = {('Allihanna', 'simbolo'): {'totemico'}}


def conteudo(texto: str) -> set:
    return {p for p in normaliza_frase(texto).split() if p not in VAZIAS}


def le_a_tabela() -> dict:
    """A Tabela 1-20 → {chave do deus: (energia, [poderes])}.

    A célula de poderes QUEBRA em duas linhas quando a lista é longa (Aharadak,
    Hyninn), e a continuação não tem nome na coluna da esquerda. Ela pertence à
    linha de cima, e é o y que diz isso.
    """
    celulas = [(x, y, t) for x, y, t in linhas_com_coordenada(PAGINA_DA_TABELA)
               if y < TABELA_ACABA_EM_Y]
    linhas: list[tuple[float, list]] = []
    for x, y, texto in sorted(celulas, key=lambda r: (r[1], r[0])):
        if linhas and y - linhas[-1][0] < 12:
            linhas[-1][1].append((x, texto))
        else:
            linhas.append((y, [(x, texto)]))
    fora = {}
    for _y, cels in linhas:
        cels.sort()
        if len(cels) < 3:
            continue
        nome, energia = cels[0][1], cels[1][1]
        poderes = ' '.join(t for _x, t in cels[2:])
        if nome in ('Divindade',):
            continue
        fora[chave(nome)] = (energia.strip().rstrip('.').lower(),
                             [p.strip() for p in poderes.split(',') if p.strip()])
    return fora


def blocos_do_capitulo(nomes: dict) -> tuple[list, list[str]]:
    """(deus, texto do verbete, página do livro) para cada bloco, em ordem de leitura."""
    fluxo, titulos = [], []
    for pagina in range(PRIMEIRA, ULTIMA + 1):
        ordenadas = sorted(linhas_com_coordenada(pagina), key=lambda r: (r[0] // 200, r[1]))
        for x, y, texto in sem_lixo(ordenadas):
            if pagina == PAGINA_DA_TABELA and y < TABELA_ACABA_EM_Y:
                continue
            if len(texto) <= TITULO_MAXIMO and chave(texto) in nomes:
                titulos.append((len(fluxo), nomes[chave(texto)], pagina - OFFSET_DO_PDF))
            fluxo.append(texto)

    aberturas = [i for i, linha in enumerate(fluxo) if linha.startswith(ABRE_O_VERBETE)]
    blocos, queixas = [], []
    for i, fim in zip(aberturas, aberturas[1:] + [len(fluxo)]):
        corpo = junta(fluxo[i:fim])
        if FECHA_O_VERBETE not in corpo:
            # A seção "Características dos deuses" EXPLICA os rótulos e não tem
            # Obrigações. Cortar até o PRÓXIMO `Crenças` é o que a distingue:
            # fatiar até o fim do capítulo faria todo bloco conter o rótulo de
            # alguém, e a seção de explicação passaria por um deus sem nome.
            continue
        antes = [x for x in titulos if x[0] < i]
        if not antes:
            queixas.append(f'bloco na linha {i} sem título de deus antes dele')
            continue
        _pos, deus, pagina = antes[-1]
        blocos.append((deus, corpo[:corpo.index(FECHA_O_VERBETE)], pagina))
    return blocos, queixas


def le_o_verbete(corpo: str) -> dict:
    lido = {}
    for i, campo in enumerate(CAMPOS):
        proximo = CAMPOS[i + 1] if i + 1 < len(CAMPOS) else FECHA_O_VERBETE
        m = re.search(re.escape(campo) + r'\. (.+?)(?: ' + re.escape(proximo) + r'\.|$)',
                      corpo, re.S)
        lido[campo] = m.group(1).strip().rstrip('.') if m else None
    return lido


def vocabulario_de_devoto() -> dict:
    """Os termos que uma lista de "Devotos" pode usar → forma canônica.

    Vocabulário FECHADO: raça, classe, e os poucos termos que o livro usa sem
    ser nenhum dos dois. O que não estiver aqui o auditor REPROVA — uma lista
    de proibidos subconta em silêncio, e esta é a inversão dela.
    """
    termos = {}
    for raca in json.load(open(DADOS / 'races.json', encoding='utf-8')).values():
        termos[chave(raca['name'])] = raca['name']
    for classe in json.load(open(DADOS / 'classes.json', encoding='utf-8')):
        termos[chave(classe['name'])] = classe['name']
    # Plural: o livro lista "elfos", o catálogo guarda "Elfo".
    for k, v in list(termos.items()):
        termos[k + 's'] = v          # elfo -> elfos
        termos[k + 'es'] = v         # caçador -> caçadores, inventor -> inventores
        termos[k.removesuffix('ao') + 'oes'] = v  # ladrão -> ladrões
    # Os que não são raça nem classe do catálogo: as metades do suraggel, o povo
    # do Thwor, e as duas aberturas ("quaisquer", "aventureiros").
    for extra in ('aggelus', 'sulfure', 'duyshidakk', 'quaisquer', 'aventureiros',
                  'golens', 'hynne', 'kliren', 'trogs', 'medusas', 'sereias', 'tritoes',
                  'silfides', 'osteon', 'dahllan', 'qareen'):
        termos.setdefault(extra, extra)
    return termos


def termos_de(texto: str, vocabulario: dict) -> tuple[set, set]:
    """(termos reconhecidos, palavras que não são termo conhecido)."""
    reconhecidos, estranhas = set(), set()
    for palavra in normaliza_frase(texto).split():
        if palavra in vocabulario:
            reconhecidos.add(vocabulario[palavra])
        elif palavra not in VAZIAS:
            estranhas.add(palavra)
    return reconhecidos, estranhas


RE_DISPONIVEL = re.compile(
    r'deus disponível para (druidas|paladinos) \(([^)]+)\)')
PAGINAS_APOS_A_CLASSE = 3  # a regra de devoção cai na abertura da classe


def deuses_por_classe() -> tuple[dict, list[str]]:
    """{'druidas': {…}, 'paladinos': {…}} lido da frase que a CLASSE imprime.

    A elegibilidade NÃO se deriva da lista de "Devotos" do deus, e a Valkaria é
    a prova: ela aceita "membros de todas as classes", mas a p61 diz que o
    druida se torna devoto "de um deus disponível para druidas (Allihanna,
    Megalokk ou Oceano)". Quem restringe é a CLASSE, e ela imprime a lista
    inteira numa frase — âncora melhor que qualquer derivação.

    As páginas saem do `classes.json`, e não de um número escrito aqui: a regra
    cai na abertura da classe, a poucas páginas do começo do capítulo dela.
    """
    classes = {c['name']: c['bookPage']
               for c in json.load(open(DADOS / 'classes.json', encoding='utf-8'))}
    fora, queixas = {}, []
    for classe, quem in (('Druida', 'druidas'), ('Paladino', 'paladinos')):
        inicio = classes[classe] + OFFSET_DO_PDF
        achou = False
        for pagina in range(inicio, inicio + PAGINAS_APOS_A_CLASSE):
            texto = junta([t for _x, _y, t in sem_lixo(
                sorted(linhas_com_coordenada(pagina), key=lambda r: (r[0] // 200, r[1])))])
            if m := RE_DISPONIVEL.search(texto):
                fora[quem] = {n.strip() for n in re.split(r',| ou ', m.group(2))}
                achou = True
                break
        if not achou:
            queixas.append(f'não achei a frase "deus disponível para {quem}" nas páginas '
                           f'{classes[classe]}-{classes[classe] + PAGINAS_APOS_A_CLASSE - 1} '
                           f'— {quem} ficaram sem medição')
    return fora, queixas


def confere(deus: dict, verbete: dict, pagina: int, vocabulario: dict,
            estranhas: set, por_classe: dict) -> list[str]:
    fora = []
    energia = (verbete['Canalizar Energia'] or '').lower()
    if deus['energia'] != energia:
        fora.append(f'energia: catálogo {deus["energia"]!r} × livro {energia!r}')

    poderes = [p.strip() for p in (verbete['Poderes Concedidos'] or '').split(',') if p.strip()]
    if deus['poderesConcedidos'] != poderes:
        fora.append(f'poderes: catálogo {deus["poderesConcedidos"]} × livro {poderes}')

    if deus['bookPage'] != pagina:
        fora.append(f'bookPage: catálogo {deus["bookPage"]} × livro {pagina}')

    arma_no_livro = verbete['Arma Preferida'] or ''
    if arma_no_livro.lower().startswith(SEM_ARMA):
        if deus.get('armaPreferida'):
            fora.append(f'armaPreferida: catálogo {deus["armaPreferida"]!r} × livro '
                        f'"não há" (o deus não tem arma preferida)')
    elif sobra := conteudo(deus.get('armaPreferida') or '') - conteudo(arma_no_livro):
        fora.append(f'armaPreferida: o catálogo diz {sorted(sobra)}, que o livro não diz '
                    f'({arma_no_livro!r})')

    sobra = conteudo(deus['simbolo']) - conteudo(verbete['Símbolo Sagrado'] or '')
    if sobra and sobra != COMPRESSAO_ACEITA.get((deus['name'], 'simbolo')):
        fora.append(f'simbolo: o catálogo diz {sorted(sobra)}, que o livro não diz '
                    f'({verbete["Símbolo Sagrado"]!r})')

    no_livro, nao_sei = termos_de(verbete['Devotos'] or '', vocabulario)
    estranhas |= nao_sei
    no_catalogo, nao_sei_cat = termos_de(' '.join(deus['devotos']), vocabulario)
    estranhas |= nao_sei_cat
    if no_livro != no_catalogo:
        fora.append(f'devotos: só no catálogo {sorted(no_catalogo - no_livro)}, '
                    f'só no livro {sorted(no_livro - no_catalogo)}')

    for campo, quem in (('paladinoEligible', 'paladinos'), ('druidaEligible', 'druidas')):
        se_pode = por_classe.get(quem)
        if se_pode is None:
            continue  # a frase da classe não foi achada; já saiu como NÃO MEDIDO
        no_livro_pode = deus['name'] in se_pode
        if deus[campo] != no_livro_pode:
            fora.append(f'{campo}: catálogo {deus[campo]} × livro {no_livro_pode} '
                        f'(a classe lista {sorted(se_pode)})')
    return fora


def main() -> int:
    gods = json.load(open(DADOS / 'gods.json', encoding='utf-8'))
    por_chave = {chave(g['name']): g['name'] for g in gods}
    por_nome = {g['name']: g for g in gods}
    vocabulario = vocabulario_de_devoto()

    tabela = le_a_tabela()
    blocos, queixas = blocos_do_capitulo(por_chave)
    por_classe, queixas_de_classe = deuses_por_classe()
    queixas += queixas_de_classe

    # CONTROLE da ligação: um deus por bloco, e a ordem tem de sair alfabética.
    ligados = [deus for deus, _corpo, _p in blocos]
    if len(set(ligados)) != len(ligados):
        queixas.append(f'algum deus ficou com dois blocos: {ligados}')
    if ligados != sorted(ligados):
        queixas.append(f'a ordem dos blocos não saiu alfabética — a geometria errou: {ligados}')

    brigas = 0
    for deus, corpo, _pagina in blocos:
        do_verbete = le_o_verbete(corpo)
        da_tabela = tabela.get(chave(deus))
        if da_tabela is None:
            queixas.append(f'{deus}: sem linha na Tabela 1-20')
            continue
        energia_tab, poderes_tab = da_tabela
        if energia_tab != (do_verbete['Canalizar Energia'] or '').lower():
            print(f'  INSTRUMENTOS DISCORDAM {deus}.energia: tabela {energia_tab!r} '
                  f'× verbete {do_verbete["Canalizar Energia"]!r}')
            brigas += 1
        poderes_verb = [p.strip() for p in (do_verbete['Poderes Concedidos'] or '').split(',')]
        if poderes_tab != poderes_verb:
            print(f'  INSTRUMENTOS DISCORDAM {deus}.poderes: tabela {poderes_tab} '
                  f'× verbete {poderes_verb}')
            brigas += 1

    divergem, estranhas = 0, set()
    for deus, corpo, pagina in blocos:
        for queixa in confere(por_nome[deus], le_o_verbete(corpo), pagina,
                              vocabulario, estranhas, por_classe):
            print(f'  {deus}: {queixa}')
            divergem += 1

    for queixa in queixas:
        print(f'  NÃO MEDIDO {queixa}')
    for (nome, campo), palavras in COMPRESSAO_ACEITA.items():
        print(f'  COMPRESSÃO ACEITA {nome}.{campo}: {sorted(palavras)} resume o que o '
              f'livro diz por extenso')
    print('  NÃO MEDIDO major: é `true` nos vinte, e o livro não separa maior de menor')
    print('  NÃO MEDIDO portfolio: é resumo de três palavras sobre o parágrafo '
          '"Crenças e Objetivos" — prosa reescrita, não transcrição')

    print(f'\ndeuses: {len(gods)} | lidos na Tabela 1-20: {len(tabela)} '
          f'| verbetes ligados: {len(blocos)} | instrumentos discordam: {brigas} '
          f'| divergem do livro: {divergem}')
    if estranhas:
        print(f'palavras da lista de Devotos que não são termo conhecido '
              f'({len(estranhas)}, para conferir que nenhuma é raça ou classe): '
              f'{sorted(estranhas)}')
    return 1 if divergem or brigas or queixas else 0


if __name__ == '__main__':
    raise SystemExit(main())
