#!/usr/bin/env python3
"""Confere os poderes GERAIS e os da TORMENTA contra a seção "Poderes" (p124-137).

Seção 6 da ALE-391. São 90 verbetes — 68 em `general-powers.json` e 22 em
`tormenta-powers.json` —, e a razão de auditá-los é que desde a ALE-401 eles são
a FONTE ÚNICA: 49 benefícios de origem e de classe APONTAM para cá em vez de
copiar a regra. A ALE-405 provou que as cópias estavam trocadas em 27 de 35
poderes únicos de origem; ninguém tinha conferido os alvos.

A faixa p124-137 guarda quatro famílias e só duas são daqui. Os Poderes
Concedidos (p127 e p132-136) são DOIS catálogos sobre as mesmas páginas e têm
issue própria, a ALE-408.

As duas âncoras, e por que o livro as dá
----------------------------------------
Como em Origens, o livro declara o mesmo dado duas vezes — aqui, os
PRÉ-REQUISITOS: a tabela de cada grupo e a linha `Pré-requisito:` do verbete. Os
dois instrumentos se conferem um ao outro ANTES de encostar no catálogo.

A razão de a segunda âncora existir, e o desenho de ler a tabela primeiro
retirando as células dela da busca por verbete, são do `engine-go/CLAUDE.md`, na
seção dos auditores. Aqui fica o que é específico desta seção, e cada armadilha
está no cabeçalho da função que a desarma.

O denominador
-------------
Por poder: se ele ancorou nas duas declarações, numa só, ou em nenhuma. E o total
de PRÉ-REQUISITOS comparados — não só de poderes —, porque um poder que ancorou e
cujo pré-requisito não foi lido sai verde sobre nada.

Uso: `python3 scripts/audit-powers.py`. Precisa do PDF do livro (veja t20pdf).
Ele PROPÕE: nada é escrito, e cada correção se revisa contra a página citada.
"""
import json
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from t20pdf import (  # noqa: E402
    RAIZ, blocos_de_tabela, celulas_da_tabela, chave, cobertura, junta, leitura,
    linhas_do_par, normaliza_frase)

PRIMEIRA, ULTIMA = 130, 143  # PDF; o livro abre a seção na p124 e fecha na p137
OFFSET_DO_PDF = 6
GERAIS = RAIZ / 'engine-go/domain/catalog/data/general-powers.json'
DA_TORMENTA = RAIZ / 'engine-go/domain/catalog/data/tormenta-powers.json'
PERICIAS = RAIZ / 'engine-go/domain/catalog/data/expertises.json'

# As páginas que têm TABELA. A da p127 é dos Poderes Concedidos (ALE-408) e não
# se lê aqui, mas as células dela saem da busca por verbete do mesmo jeito.
PAGINAS_DE_TABELA = (132, 133, 134)
# A COBERTURA AQUI SE MEDE AO CONTRÁRIO da que nasceu nas origens, e o motivo é
# o terreno: lá o catálogo escrevia uma forma curta EQUIVALENTE à do livro, e a
# pergunta era "a regra do livro está aqui?". Aqui ele RESUME —
# "+2 nos ataques com a arma escolhida" para um parágrafo inteiro —, e medir
# livro→catálogo dá 20% para a Torcida, cuja regra está perfeita.
#
# A pergunta certa para um resumo é a inversa: **tudo o que o catálogo AFIRMA
# está no livro?** Um resumo fiel só usa palavras que a página tem; uma regra
# inventada traz palavras que não estão lá. Medido, a direção invertida põe 84
# dos 88 acima de 60% e isola os quatro que merecem o olho.
#
# O piso é mais frouxo que o das origens de propósito, e não há vão limpo nesta
# distribuição: ele ORDENA a lista para a revisão humana em vez de decidir
# sozinho. O veredito sobre prosa continua sendo do olho.
COBERTURA_MINIMA = 0.5

# O QUE FICA ACEITO, e cada linha diz por quê. Declarar é o que faz um caso NOVO
# reprovar com o NOME dele, em vez de se somar a uma lista que já era vermelha e
# que ninguém mais lê. Uma linha que PARAR de valer também reprova.
ACEITO = {
    'Foco em Arma': 'o catálogo diz "pego várias vezes" onde o livro diz "escolher '
                    'este poder outras vezes" (p128) — sinônimo, e o +2 e o escopo '
                    'batem',
    'Foco em Perícia': 'o catálogo escreve "2d20" e a p130 escreve "dois dados". Em '
                       'T20 teste de perícia É d20, então o catálogo especifica o que '
                       'a página deixa implícito',
    'Inventário Organizado': 'o catálogo escreve "1/2" e a p130 escreve "meio '
                             'espaço" — o mesmo número por extenso, que a tradução de '
                             'numeral não alcança por ser fração',
    'Ao Sabor do Destino': 'a progressão por patamar está na tabela da página '
                           'SEGUINTE (p130), fora do verbete — o catálogo a transcreve '
                           'e o auditor não a lê. NÃO MEDIDO, não aprovado',
    'Disparo Preciso': 'o catálogo aceita "Estilo de Disparo OU Estilo de Arremesso" e '
                       'o VERBETE da p125 diz exatamente isso; quem simplifica é a '
                       'TABELA da p126, que a indentação obriga a um pai só',
}

# ONDE O LIVRO DISCORDA DE SI MESMO. Não é defeito nosso, e o catálogo segue o
# VERBETE — que é onde a regra é enunciada por extenso.
DISCORDANCIA_DECLARADA = {
    'Estilo de Arma e Escudo': 'a tabela da p126 não lista "treinado em Luta" e o '
                               'verbete da p125 exige',
    'Disparo Preciso': 'a tabela da p126 lista só "Estilo de Disparo" e o verbete da '
                       'p125 aceita "ou Estilo de Arremesso"',
}


def le_a_tabela():
    """Âncora 1: `{nome: (pré-requisito, nível, pai)}` das tabelas de p126 e p128.

    O `pai` é o poder acima na árvore — a indentação é o que diz "o pré-requisito
    deste é aquele", e o `—` de uma linha indentada significa "nenhum ALÉM do
    pai", não "nenhum".
    """
    da_tabela, raizes = {}, {}
    for pagina in (132, 134):
        for pares, corpo in blocos_de_tabela(pagina):
            for x_poder, x_pre in pares:
                linhas, _consumidas = linhas_do_par(corpo, x_poder, x_pre)
                for nome, pre, nivel in linhas:
                    raizes[nivel] = nome
                    pai = raizes.get(nivel - 1) if nivel else None
                    da_tabela[chave(nome)] = (nome, pre, nivel, pai)
    return da_tabela


def le_os_verbetes(nomes: list[str]):
    """Âncora 2: `{chave: (nome, página do livro, corpo)}`.

    O nome aparece uma TERCEIRA vez no capítulo, como pré-requisito de outro
    poder, e quando a quebra de linha o deixa sozinho ele tem cara de título. O
    discriminante está na PRÓPRIA linha: um título não termina em ponto, e a
    menção termina, porque ela fecha a frase ("Pré-requisito:" / "Estilo de Duas
    Armas.").

    Olhar a linha ANTERIOR não serve, e foi a primeira tentativa: ela quase
    sempre termina com o `Pré-requisito: …` do verbete de cima, e o filtro
    descartou 26 verbetes legítimos.
    """
    por_chave = {chave(n): n for n in nomes}
    linhas = []
    for pagina in range(PRIMEIRA, ULTIMA + 1):
        consumidas = celulas_da_tabela(pagina) if pagina in PAGINAS_DE_TABELA else set()
        for x, y, texto in leitura(pagina):
            if (round(x, 1), round(y, 1)) not in consumidas:
                linhas.append((pagina, texto))

    achados = []
    for i, (pagina, texto) in enumerate(linhas):
        if chave(texto) in por_chave and not texto.rstrip().endswith('.'):
            achados.append((i, pagina, por_chave[chave(texto)]))

    verbetes, vistos = {}, {}
    for n, (i, pagina, nome) in enumerate(achados):
        fim = achados[n + 1][0] if n + 1 < len(achados) else len(linhas)
        vistos[nome] = vistos.get(nome, 0) + 1
        if chave(nome) not in verbetes:
            verbetes[chave(nome)] = (nome, pagina - OFFSET_DO_PDF,
                                     junta([t for _p, t in linhas[i:fim]]))
    return verbetes, vistos


# O VOCABULÁRIO DA COLUNA DE PRÉ-REQUISITOS, levantado do próprio livro: são 45
# textos distintos nas duas tabelas, e todos caem numa destas formas.
ATRIBUTOS = {'For': 'strength', 'Des': 'dexterity', 'Con': 'constitution',
             'Int': 'intelligence', 'Sab': 'wisdom', 'Car': 'charisma'}
RE_ATRIBUTO = re.compile(rf'^({"|".join(ATRIBUTOS)})\s+(\d+)$')
RE_NIVEL = re.compile(r'^(\d+)º nível de personagem$')
# `poderes?` exigiria "podere": o `?` cobre só o `s`. A coluna escreve
# "Um poder da Tormenta" no singular, e o grupo é o que a torna opcional.
RE_TORMENTA = re.compile(r'^(Um|Dois|Três|Quatro|Cinco)\s+poder(?:es)?\s+da\s+Tormenta$')
QUANTOS = {'Um': 1, 'Dois': 2, 'Três': 3, 'Quatro': 4, 'Cinco': 5}
RE_TREINADO = re.compile(r'^treinad[oa]\s+(?:em\s+|n[ao]\s+)?(.+)$', re.I)
# O ponto final pode ser o ÚLTIMO caractere do verbete — exigir ponto MAIS
# espaço faz o último verbete de cada trecho sair sem pré-requisito, e o
# relatório acusa o livro de discordar de si mesmo em 56 poderes.
RE_PRE_DO_VERBETE = re.compile(r'Pré-?\s*-?requisitos?:\s*(.+?)\.(?=\s|$)')
SEM_PRE = '—'


def um_pre_requisito(pedaco: str, pericias: set[str], poderes: set[str]) -> str:
    """Um pedaço da coluna vira a forma normalizada que o catálogo também produz."""
    pedaco = pedaco.strip()
    if not pedaco or pedaco == SEM_PRE:
        return ''
    if m := RE_ATRIBUTO.match(pedaco):
        return f'attr:{ATRIBUTOS[m.group(1)]}:{m.group(2)}'
    if m := RE_NIVEL.match(pedaco):
        return f'level:{m.group(1)}'
    if m := RE_TORMENTA.match(pedaco):
        return f'tormenta:{QUANTOS[m.group(1)]}'
    if m := RE_TREINADO.match(pedaco):
        pedaco = m.group(1).strip()
    if chave(pedaco) in pericias:
        return f'trained:{chave(pedaco)}'
    # A célula pode trazer o NOME de um poder em vez do travessão: a "Carga de
    # Cavalaria" está indentada sob "Ginete" e a célula dela diz "Ginete".
    if chave(pedaco) in poderes:
        return f'power:{chave(pedaco)}'
    return f'nota:{normaliza_frase(pedaco)}'


def le_pre_requisitos(texto: str, pericias: set[str], poderes: set[str],
                      pai: str | None) -> set[str]:
    """A coluna inteira, que pode trazer vários separados por vírgula.

    A vírgula NÃO separa dentro de parênteses: "Ofício (alquimista)" é um
    requisito só, e "Habilidade Magias, Ofício (escriba)" são dois.
    """
    pedacos, nivel, atual = [], 0, ''
    for c in texto:
        if c == '(':
            nivel += 1
        elif c == ')':
            nivel -= 1
        if c == ',' and nivel == 0:
            pedacos.append(atual)
            atual = ''
            continue
        atual += c
    pedacos.append(atual)
    lidos = {um_pre_requisito(p, pericias, poderes) for p in pedacos}
    # A INDENTAÇÃO implica o pai SEMPRE, e não só quando a célula traz `—`. O
    # travessão diz "nada além do pai"; um texto diz "isto E o pai" — o
    # "Fanático" está indentado sob "Encouraçado" com "12º nível de personagem",
    # e o catálogo confirma: `minLevel 12` mais `{kind: power, id: encouracado}`.
    if pai:
        lidos.add(f'power:{chave(pai)}')
    return {x for x in lidos if x}


def do_catalogo(poder: dict, por_uid: dict, por_id: dict) -> set[str]:
    """Os pré-requisitos do catálogo na MESMA forma normalizada."""
    fora = set()
    if poder.get('minLevel'):
        fora.add(f'level:{poder["minLevel"]}')
    if poder.get('requiresOtherPowers'):
        fora.add(f'tormenta:{poder["requiresOtherPowers"]}')
    if poder.get('requiresPower'):
        fora.add(f'power:{chave(por_id.get(poder["requiresPower"], ""))}')
    for regra in poder.get('prerequisites') or []:
        tipo = regra.get('kind')
        if tipo == 'attribute':
            fora.add(f'attr:{regra["attr"]}:{regra["min"]}')
        elif tipo == 'trained':
            fora.add(f'trained:{chave(regra["expertise"])}')
        elif tipo == 'power':
            fora.add(f'power:{chave(por_id.get(regra["id"], ""))}')
        elif tipo == 'anyPower':
            fora.add('power:' + '|'.join(sorted(chave(por_id.get(i, ''))
                                                for i in regra['ids'])))
        else:
            fora.add(f'nota:{normaliza_frase(regra.get("description", ""))}')
    return fora


def pericias_citadas(texto: str, nomes: list[str]) -> set[str]:
    return {n for n in nomes if re.search(rf'\b{re.escape(n)}\b', texto)}


RE_NUMERO = re.compile(r'[+\-–−]?\d+(?:\.\d{3})*')
# A REMISSÃO DE PÁGINA não é número de regra, e ela aparece nas DUAS formas:
# o livro escreve "veja a página 260" e "as páginas 333 e 341", o catálogo
# escreve "ver p260". Filtrar só uma das formas deixa a assimetria acusando
# um verbete cuja regra está certa.
RE_REMISSAO = re.compile(r'p(?:[áa]g(?:ina|s?\.)?)?\s*\d+(?:\s+e\s+\d+)?')


# O LIVRO ESCREVE O NUMERAL POR EXTENSO e o catálogo em dígito: "uma ação de
# movimento extra" × "1 ação", "Uma vez por rodada" × "1×/rodada". Sem esta
# tradução, cinco poderes cuja regra está certa aparecem como número inventado —
# e o `1` era o mais frequente de todos.
POR_EXTENSO = {'um': '1', 'uma': '1', 'dois': '2', 'duas': '2', 'três': '3',
               'quatro': '4', 'cinco': '5', 'seis': '6', 'sete': '7', 'oito': '8',
               'nove': '9', 'dez': '10'}
RE_EXTENSO = re.compile(r'\b(' + '|'.join(POR_EXTENSO) + r')\b', re.I)


def numeros(texto: str) -> set[str]:
    """Os números que a frase imprime. A remissão de página não é número de
    regra, e contá-la acusaria um verbete cuja descrição está certa."""
    limpo = RE_EXTENSO.sub(lambda m: POR_EXTENSO[m.group(1).lower()],
                           RE_REMISSAO.sub(' ', texto))
    return {n.translate({ord(c): '-' for c in '–−'}).lstrip('+')
            for n in RE_NUMERO.findall(limpo)}


def regra_do_verbete(corpo: str, nome: str) -> str:
    """O texto do verbete sem o título e sem a linha de pré-requisito — só a
    regra, que é o que o catálogo descreve."""
    _cabeca, _sep, resto = corpo.partition(nome)
    return RE_PRE_DO_VERBETE.split(resto)[0].strip() if resto else ''


def so_a_forma(pre: set[str]) -> set[str]:
    """Os pré-requisitos que têm FORMA, sem as notas em prosa.

    A tabela e o verbete dizem a mesma coisa com palavras diferentes — a coluna
    escreve "Armaduras pesadas" e o verbete "Proficiência com armaduras pesadas";
    "Habilidade Magias" e "Habilidade de classe Magias". Comparar a prosa produz
    uma discordância por redação a cada dois poderes, e nenhuma delas é sobre a
    regra.

    O que tem forma — atributo, nível, perícia treinada, poder, contagem da
    Tormenta — se compara por igualdade. A nota sai listada para o OLHO, como a
    "compressão aceita" do auditor de deuses.
    """
    return {r for r in pre if not r.startswith('nota:')}


def confere(poder: dict, da_tabela, do_verbete, pagina: int | None, regra: str,
            pericias: list[str], medidos: dict) -> tuple[list[str], list[str], list[str]]:
    """As queixas deste poder, onde o LIVRO discorda de si, e as notas a reler."""
    fora, livro, notas = [], [], []
    if da_tabela is not None and do_verbete is not None:
        if so_a_forma(da_tabela) != so_a_forma(do_verbete):
            if poder['name'] not in DISCORDANCIA_DECLARADA:
                livro.append(f'{poder["name"]}: a tabela diz '
                             f'{sorted(so_a_forma(da_tabela))} e o verbete diz '
                             f'{sorted(so_a_forma(do_verbete))} — discordância NOVA')
        elif poder['name'] in DISCORDANCIA_DECLARADA:
            livro.append(f'{poder["name"]}: a discordância declarada NÃO acontece mais '
                         f'— tire a linha de DISCORDANCIA_DECLARADA')
        elif da_tabela != do_verbete:
            notas.append(f'{poder["name"]}: tabela {sorted(da_tabela - do_verbete)} × '
                         f'verbete {sorted(do_verbete - da_tabela)}')
    lido = do_verbete if do_verbete else da_tabela
    if lido is not None:
        medidos['pre'] += len(so_a_forma(lido)) or 1
        if so_a_forma(lido) != so_a_forma(poder['_pre']):
            fora.append(f'pré-requisitos: catálogo {sorted(so_a_forma(poder["_pre"]))} × '
                        f'livro {sorted(so_a_forma(lido))}')
        elif lido != poder['_pre']:
            notas.append(f'{poder["name"]}: catálogo {sorted(poder["_pre"] - lido)} × '
                         f'livro {sorted(lido - poder["_pre"])}')
    if pagina is not None:
        medidos['pagina'] += 1
        if poder.get('bookPage') != pagina:
            fora.append(f'bookPage: catálogo {poder.get("bookPage")} × livro {pagina}')
    if regra:
        fora += confere_a_regra(poder, regra, pericias, medidos)
    return fora, livro, notas


def confere_a_regra(poder: dict, do_livro: str, pericias: list[str],
                    medidos: dict) -> list[str]:
    """As três dimensões sobre a descrição: números, perícias nomeadas, e quanto
    das palavras de conteúdo do livro o catálogo repete.

    A prosa não se compara — o catálogo encurta de propósito. As três se comparam
    porque uma forma mais curta as PRESERVA, e a terceira existe porque as duas
    primeiras deixaram passar seis regras trocadas na ALE-405.
    """
    do_catalogo = poder.get('description', '')
    medidos['regra'] += 1
    fora = []
    # AS TRÊS DIMENSÕES MEDEM NA MESMA DIREÇÃO: o que o CATÁLOGO afirma tem de
    # estar na página. Aqui ele RESUME — o livro traz exemplo trabalhado,
    # progressão e remissão, e o catálogo fica com a regra —, então exigir que o
    # catálogo repita tudo acusaria quase todos. O que não se admite é o
    # contrário: um número ou uma perícia que o catálogo afirma e a página não
    # tem é regra inventada.
    inventados = numeros(do_catalogo) - numeros(do_livro)
    if inventados:
        fora.append(f'números que o catálogo afirma e a página não tem: '
                    f'{sorted(inventados)}')
    no_catalogo = pericias_citadas(do_catalogo, pericias)
    de_fora = no_catalogo - pericias_citadas(do_livro, pericias)
    if de_fora:
        fora.append(f'perícias que o catálogo cita e a página não: {sorted(de_fora)}')
    razao, juntas, total = cobertura(do_catalogo, do_livro)
    medidos['cobertura'].append((razao, poder['name']))
    if razao < COBERTURA_MINIMA:
        fora.append(f'{juntas} das {total} palavras de conteúdo do CATÁLOGO estão no '
                    f'livro ({razao:.0%}, piso {COBERTURA_MINIMA:.0%}) — um resumo fiel '
                    f'só usa palavras que a página tem')
    if fora:
        fora.append(f'  NO LIVRO:    {do_livro[:260]!r}')
        fora.append(f'  NO CATÁLOGO: {do_catalogo[:260]!r}')
    return fora


def main() -> int:
    gerais = json.load(open(GERAIS, encoding='utf-8'))
    tormenta = json.load(open(DA_TORMENTA, encoding='utf-8'))
    tormenta = tormenta if isinstance(tormenta, list) else list(tormenta.values())
    for p in gerais:
        p['_familia'] = 'geral'
    for p in tormenta:
        p['_familia'] = 'tormenta'
    todos = gerais + tormenta
    nomes_de_pericia = [p['name'] for p in json.load(open(PERICIAS, encoding='utf-8'))]
    pericias = {chave(n) for n in nomes_de_pericia}
    poderes = {chave(p['name']) for p in todos}
    por_id = {p['id']: p['name'] for p in todos}
    for p in todos:
        p['_pre'] = do_catalogo(p, {}, por_id)

    da_tabela = le_a_tabela()
    verbetes, vistos = le_os_verbetes([p['name'] for p in todos])

    medidos = {'pre': 0, 'pagina': 0, 'regra': 0, 'cobertura': []}
    divergem, nas_duas, numa_so = 0, 0, 0
    em_nenhuma, do_livro, queixas, notas = [], [], [], []
    aceitos = set()
    for poder in todos:
        k = chave(poder['name'])
        linha = da_tabela.get(k)
        tabela = (le_pre_requisitos(linha[1], pericias, poderes, linha[3])
                  if linha else None)
        verbete = verbetes.get(k)
        pagina, corpo = (verbete[1], verbete[2]) if verbete else (None, '')
        m = RE_PRE_DO_VERBETE.search(corpo) if corpo else None
        if poder['_familia'] == 'tormenta':
            # O VERBETE DA TORMENTA NÃO DECLARA PRÉ-REQUISITO: a seção os imprime
            # só na tabela da p128. Ler a ausência como "nenhum" faria os 22
            # discordarem da tabela, e a discordância seria do instrumento.
            do_verbete = None
        elif m:
            do_verbete = le_pre_requisitos(m.group(1), pericias, poderes, None)
        else:
            do_verbete = set() if corpo else None
        if tabela is None and do_verbete is None:
            em_nenhuma.append(f'{poder["name"]}: não ancorou em nenhuma das duas '
                              f'declarações — NADA dele foi medido')
            continue
        if tabela is None or do_verbete is None:
            numa_so += 1
            queixas.append(f'{poder["name"]}: não ancorou '
                           f'{"na tabela" if tabela is None else "no verbete"}')
        else:
            nas_duas += 1
        fora, discorda, prosa = confere(poder, tabela, do_verbete, pagina,
                                        regra_do_verbete(corpo, poder['name']),
                                        nomes_de_pericia, medidos)
        do_livro += discorda
        notas += prosa
        if poder['name'] in ACEITO:
            aceitos.add(poder['name'])
            continue
        for queixa in fora:
            print(f'  {poder["name"]}: {queixa}' if not queixa.startswith('  ')
                  else f'  {queixa}')
            divergem += not queixa.startswith('  ')

    if do_livro:
        print('\nO LIVRO DISCORDA DE SI MESMO (achado sobre o LIVRO):')
        for linha in do_livro:
            print(f'  {linha}')
    if notas:
        print(f'\nPRÉ-REQUISITO QUE SÓ DIFERE NA REDAÇÃO ({len(notas)}, para o OLHO — a '
              f'forma bate):')
        for linha in notas:
            print(f'  {linha}')
    for linha in em_nenhuma + queixas:
        print(f'  NÃO MEDIDO {linha}')
    repetidos = {k: v for k, v in vistos.items() if v > 1}
    if repetidos:
        print(f'  NÃO MEDIDO títulos achados mais de uma vez: {repetidos}')

    for nome, motivo in ACEITO.items():
        print(f'  ACEITO {nome}: {motivo}')
        if nome not in aceitos:
            print(f'  PROBLEMA {nome} está em ACEITO e não diverge mais — tire a linha')
            divergem += 1
    for nome, motivo in DISCORDANCIA_DECLARADA.items():
        print(f'  DISCORDÂNCIA DECLARADA {nome}: {motivo}')

    coberturas = medidos.pop('cobertura')
    print(f'\npoderes: {len(todos)} | ancoraram nas DUAS declarações: {nas_duas} '
          f'| em UMA só: {numa_so} | em NENHUMA (≠ corretos): {len(em_nenhuma)}')
    print(f'lidos na tabela: {len(da_tabela)}/{len(todos)} | '
          f'verbetes achados: {len(verbetes)}/{len(todos)}')
    print('COMPARADOS: ' + ', '.join(f'{c} {n}' for c, n in medidos.items()))
    print(f'cobertura da regra, da pior para a melhor ({len(coberturas)} medidas, '
          f'piso {COBERTURA_MINIMA:.0%}):')
    for razao, nome in sorted(coberturas)[:14]:
        print(f'  {razao:5.0%} {nome}')
    print(f'divergem do livro: {divergem} | o livro discorda de si: {len(do_livro)}')
    return 1 if divergem or do_livro or em_nenhuma else 0


if __name__ == '__main__':
    raise SystemExit(main())
