#!/usr/bin/env python3
"""Confere os dois catálogos de origem contra a seção "Origens" (p85-95). Seção 3 da ALE-391.

O que ele mede
--------------
A LISTA DE BENEFÍCIOS de cada origem — perícias, poderes gerais, poder único e a
categoria do poder "a sua escolha" —, mais a linha de itens e a página. E mede os
dois catálogos UM CONTRA O OUTRO: `origins-source.json` (a transcrição do livro,
por slug) e `origins.json` (a mesma origem achatada em benefícios) descrevem as
MESMAS 35 origens e compartilham o `uid`, então uma divergência entre eles é
defeito por si. Essa metade não precisa do livro e desceu para Go, no
`TestEveryOriginSaysTheSameThingInBothFiles`; aqui ela fica porque o relatório
vale mais junto.

Por que são DUAS âncoras, por que a tabela é lida primeiro, e por que a descrição
do poder único se cerca por TRÊS dimensões em vez de se comparar como prosa: o
`engine-go/CLAUDE.md` é o dono dessas razões, na seção dos auditores. O que mora
aqui é o que é específico deste código, e cada armadilha está no cabeçalho da
função que a desarma.

O denominador
-------------
Por origem: em quantas das duas declarações ela ancorou. E o total de BENEFÍCIOS
comparados — não só de origens —, porque uma origem que ancorou e cujo regex de
lista não casou sai verde com zero benefícios medidos. "Verde" e "não mediu" são
a mesma cor sem denominador.

Uso: `python3 scripts/audit-origins.py`. Precisa do PDF do livro (veja t20pdf).
Ele PROPÕE: nada é escrito, e cada correção se revisa contra a página citada.
"""
import json
import re
import sys
import unicodedata
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from t20pdf import (  # noqa: E402
    RAIZ, blocos_da_pagina, chave, coluna_de, inicios_das_colunas, junta,
    linhas_com_coordenada, sem_lixo)

PRIMEIRA, ULTIMA = 91, 101  # PDF; o livro abre a seção na p85 e fecha na p95
OFFSET_DO_PDF = 6  # livro = PDF - 6
FONTE = RAIZ / 'engine-go/domain/catalog/data/origins-source.json'
ACHATADO = RAIZ / 'engine-go/domain/catalog/data/origins.json'
PERICIAS = RAIZ / 'engine-go/domain/catalog/data/expertises.json'

# A ORIGEM SEM LISTA, e ela é a exceção NOMEADA: o Amnésico não imprime
# "(perícias); … (poderes)." nenhuma das duas vezes — a linha dele é "Uma perícia
# e um poder escolhidos pelo mestre e o poder Lembranças Graduais", e o benefício
# é o mestre. Nomear a exceção é o que faz uma origem NOVA com forma nova
# reprovar em vez de nascer sem medição.
SEM_LISTA = 'amnesico'

# ONDE O LIVRO DISCORDA DE SI MESMO. A Tabela 1-19 dá ao Fazendeiro
# `Ofício (fazendeiro)` (p87) e o verbete dele dá `Ofício` sem especialização
# (p90). Não é defeito nosso: o catálogo segue a TABELA, como nas outras três
# origens com especialização — Assistente de Laboratório, Minerador e Taverneiro,
# que as DUAS declarações grafam com o parêntese.
#
# Declarar é o que faz uma discordância NOVA reprovar com o nome dela, em vez de
# se somar a uma linha que já era vermelha e que ninguém mais lê. E uma declarada
# que PARAR de discordar reprova também, senão esta tabela apodrece sozinha.
DISCORDANCIA_DECLARADA = {
    'fazendeiro': 'a Tabela 1-19 grafa "Ofício (fazendeiro)" e o verbete grafa '
                  '"Ofício" — o catálogo segue a tabela',
}

# O TÍTULO QUE FECHA A SEÇÃO. O último verbete — o Trabalhador — não tem um
# verbete seguinte para delimitá-lo, e sem isto a descrição do poder único dele
# engole a barra lateral "Sua Própria Origem" inteira: 2.696 caracteres de prosa
# entram na comparação como se fossem a regra.
FECHA_A_SECAO = 'Sua Própria Origem'

# "um poder de combate a sua escolha" e "um poder da Tormenta a sua escolha" não
# são nome de poder: o catálogo os guarda como CATEGORIA de escolha, e comparar
# a frase com a lista de nomes acusaria as cinco origens que a têm.
RE_ESCOLHA = re.compile(r'um poder (?:de |da )?(\w+) a sua escolha')
# `(poder)` no SINGULAR existe: o Amigo dos Animais lista um poder só, e um regex
# que só conhecesse o plural deixaria o verbete dele sem ancorar — em silêncio,
# porque "não casou" e "não existe" se parecem.
RE_LISTA = re.compile(
    r'Benefícios\.\s*(.*?)\s*\(perícias\);\s*(.*?)\s*\(poder(?:es)?\)\.')
RE_ITENS = re.compile(r'Itens\.\s*(.*?)\.\s')
RE_NUMERO = re.compile(r'[+\-–−]?\d+(?:\.\d{3})*')
# A REMISSÃO DE PÁGINA não é número de regra. O "Estoico" fecha com "Veja as
# regras de recuperação na página 106", e contar o 106 como número da regra
# acusaria um poder cuja descrição está certa.
RE_REMISSAO = re.compile(r'(?:p[áa]g(?:ina|s?\.)?)\s*\d+')
# `Ofício (alquimista)` é o `Ofício` do livro com a especialização entre
# parênteses. O `origins.json` guarda só a perícia, porque é ela que o motor
# treina; o `origins-source.json` guarda a frase do livro. Comparar os dois pela
# frase inteira acusaria as quatro origens que têm especialização.
RE_ESPECIALIZACAO = re.compile(r'\s*\([^)]*\)\s*$')

# As PALAVRAS VAZIAS da cobertura: artigo, preposição, pronome e verbo de ligação
# estão em toda frase, e contá-los daria a qualquer par de frases em português uma
# semelhança de base que esconderia a diferença que importa.
PALAVRAS_VAZIAS = frozenset(
    'a o as os um uma uns umas de do da dos das em no na nos nas e ou que se por '
    'para com sem seu sua seus suas voce ele ela isso este esta esse essa ao aos '
    'nao mas como mesmo ate entre sobre pelo pela mais menos tambem cada qualquer '
    'todos todas outro outra sao ser tem pode podem quando onde apenas alem disso '
    'muito seja ja la'.split())
# O PISO DE COBERTURA das palavras de conteúdo do livro, e ele foi MEDIDO, não
# escolhido: sobre os 34 poderes únicos antes da correção da ALE-405, as 27 regras
# trocadas ficaram todas em 38% ou menos e as 7 formas curtas fiéis em 67% ou
# mais. O piso mora no vão, com margem para os dois lados.
#
# Ele existe porque as outras duas dimensões NÃO bastaram: os números do "Vendedor
# de Carcaças" coincidiram por acaso — o livro dá +5 no teste e o catálogo dava
# T$ 5 por ND —, e nem ele nem o "Quebra-Galho", o "Antigo Mestre", o "Vida
# Rústica", o "Truque de Mágica" ou o "Influência Militar" citam perícia. Seis
# regras trocadas passaram verdes por duas dimensões medirem o que elas não
# tinham.
COBERTURA_MINIMA = 0.5

CABECALHO_NOME, CABECALHO_BENEFICIOS = 'Origem', 'Benefícios'
TITULO_DA_TABELA = 'Tabela 1-19: Origens'
MESMA_COLUNA = 2.0  # pontos de tolerância no x de uma célula da tabela
# A SEPARAÇÃO MÍNIMA entre duas colunas de verdade. A calha desta diagramação tem
# mais de 200 pontos; quatro não são coluna nova.
SEPARACAO_MINIMA_DE_COLUNA = 40.0



def colunas(blocos) -> list[float]:
    """Os x onde as colunas começam, com uma separação mínima entre dois inícios.

    O `inicios_das_colunas` do `t20pdf` conta a FREQUÊNCIA do x e nada mais, e na
    p87 isso devolve TRÊS colunas onde há duas: a célula de nome da Tabela 1-19
    começa em x=74,9 e a prosa da coluna começa em x=70,9 — quatro pontos de
    distância, 36 blocos contra 4, e as duas passam o piso de frequência.

    A página inteira sai fora de ordem quando isso acontece, e o sintoma não se
    parece com desordem: o `Benefícios.` do Artista aparece DEPOIS do `(poderes).`
    dele, o regex da lista não casa, e a origem sai como "não ancorou no verbete"
    — que é indistinguível de um verbete que não existe.
    """
    inicios = inicios_das_colunas(blocos)
    juntas = [inicios[0]]
    for x in inicios[1:]:
        if x - juntas[-1] >= SEPARACAO_MINIMA_DE_COLUNA:
            juntas.append(x)
    return juntas


def leitura(pagina: int) -> list[tuple[float, float, str]]:
    """As linhas da página em ordem de LEITURA, mas com o (x, y) preservado.

    O `linhas_da_pagina` do `t20pdf` dá a ordem e descarta a coordenada, e aqui
    ela é necessária duas vezes: para parear as duas colunas da Tabela 1-19 pelo
    y, e para RETIRAR as linhas da tabela da busca por verbete.

    Ordenar pelo y da LINHA (e não do bloco) é o que sobrevive à p88, onde o
    texto corre ao redor de uma ilustração e o x de cada linha muda — 'Charlatão'
    é impresso em x=359 numa coluna que começa em x=289.
    """
    blocos = list(blocos_da_pagina(pagina))
    inicios = colunas(blocos) if blocos else [0.0]
    linhas = sem_lixo(linhas_com_coordenada(pagina))
    linhas.sort(key=lambda t: (coluna_de(inicios, t[0]), t[1]))
    return linhas


def pagina_da_tabela() -> int | None:
    for pagina in range(PRIMEIRA, ULTIMA + 1):
        if any(t == TITULO_DA_TABELA for _x, _y, t in leitura(pagina)):
            return pagina
    return None


def celulas_da_tabela(pagina: int) -> tuple[dict, list[tuple[int, float, float]], list[str]]:
    """As duas colunas da Tabela 1-19 agrupadas pelo y — o que diz que um nome e
    uma lista de benefícios são a MESMA linha. Parear por ÍNDICE erraria na
    primeira linha que quebra em duas, e a do Assistente de Laboratório quebra nas
    DUAS colunas."""
    linhas = leitura(pagina)
    cabecalho = [(x, y) for x, y, t in linhas if t == CABECALHO_NOME]
    if not cabecalho:
        return {}, [], [f'PROBLEMA a coluna {CABECALHO_NOME!r} da {TITULO_DA_TABELA} não '
                        f'foi achada na p{pagina - OFFSET_DO_PDF}']
    x_nome, y_cabecalho = cabecalho[0]
    x_beneficios = next(
        (x for x, y, t in linhas
         if abs(y - y_cabecalho) < MESMA_COLUNA and t == CABECALHO_BENEFICIOS), None)
    if x_beneficios is None:
        return {}, [], [f'PROBLEMA a coluna {CABECALHO_BENEFICIOS!r} da '
                        f'{TITULO_DA_TABELA} não foi achada na '
                        f'p{pagina - OFFSET_DO_PDF}']
    celulas: dict[float, dict[str, str]] = {}
    consumidas: list[tuple[int, float, float]] = []
    for x, y, texto in linhas:
        if y <= y_cabecalho:
            continue
        lado = ('nome' if abs(x - x_nome) < MESMA_COLUNA
                else 'beneficios' if abs(x - x_beneficios) < MESMA_COLUNA else None)
        if lado is None:
            continue
        celulas.setdefault(round(y, 1), {})[lado] = texto
        consumidas.append((pagina, x, y))
    return celulas, consumidas, []


def le_tabela(nomes: dict) -> tuple[dict, list[str], list[tuple[int, float, float]]]:
    """Âncora 1: as linhas da Tabela 1-19 (p87), pareadas pelo y das duas colunas.

    Devolve `{slug: texto de benefícios}`, as notas com o denominador, e o
    (página, x, y) de cada linha consumida — que é o que a âncora 2 tem de
    ignorar, ou o rodapé da coluna dá ao "Trabalhador" o verbete do Artista.
    """
    pagina = pagina_da_tabela()
    if pagina is None:
        return {}, [f'PROBLEMA {TITULO_DA_TABELA!r} não foi achado entre as páginas '
                    f'{PRIMEIRA - OFFSET_DO_PDF}-{ULTIMA - OFFSET_DO_PDF}'], []
    celulas, consumidas, problemas = celulas_da_tabela(pagina)
    if problemas:
        return {}, problemas, []

    por_chave = {chave(nome): slug for slug, nome in nomes.items()}
    da_tabela, notas, slug = {}, [], None
    # O nome de uma linha pode ocupar DUAS linhas de texto. A que não casa sozinha
    # fica pendente, e a linha seguinte decide o que ela era: nome acumulado de
    # uma linha NOVA, ou continuação da linha de cima.
    pendente: tuple[str, str] | None = None
    for y in sorted(celulas):
        nome = celulas[y].get('nome', '')
        beneficios = celulas[y].get('beneficios', '')
        if chave(nome) in por_chave:
            slug, pendente = por_chave[chave(nome)], None
            da_tabela[slug] = beneficios
            continue
        if pendente and chave(f'{pendente[0]} {nome}') in por_chave:
            slug = por_chave[chave(f'{pendente[0]} {nome}')]
            da_tabela[slug] = f'{pendente[1]} {beneficios}'.strip()
            notas.append(f'{nomes[slug]}: o nome ocupa duas linhas da tabela '
                         f'({pendente[0]!r} + {nome!r}) — as duas metades foram '
                         f'juntadas nas DUAS colunas')
            pendente = None
            continue
        if nome:
            pendente = (nome, beneficios)
            continue
        if slug is None:
            notas.append(f'PROBLEMA célula da tabela antes da primeira origem: '
                         f'{beneficios!r}')
            continue
        da_tabela[slug] = f'{da_tabela[slug]} {beneficios}'.strip()
    if pendente:
        notas.append(f'PROBLEMA nome de linha da tabela sem origem no catálogo: '
                     f'{pendente[0]!r} / {pendente[1]!r}')
    notas.append(f'grupos de y na tabela: {len(celulas)} → linhas casadas com uma '
                 f'origem: {len(da_tabela)} (a diferença são as quebras nomeadas '
                 f'acima)')
    return da_tabela, notas, consumidas


def le_verbetes(nomes: dict, consumidas: set) -> tuple[dict, list[str]]:
    """Âncora 2: o verbete de cada origem — o título com `Benefícios.` abaixo.

    Devolve `{slug: (página do livro, [linhas do verbete])}`. A segunda condição
    é o que separa verbete de linha de tabela: "Batedor" é impresso na p87 antes
    da p88, e casar só por título acusa 29 das 35 origens com `bookPage` errado.
    """
    por_chave = {chave(nome): slug for slug, nome in nomes.items()}
    linhas: list[tuple[int, str]] = []
    for pagina in range(PRIMEIRA, ULTIMA + 1):
        for x, y, texto in leitura(pagina):
            if (pagina, x, y) in consumidas:
                continue
            linhas.append((pagina, texto))

    # Um título é uma linha que É o nome da origem, e ele pode quebrar em duas —
    # "Assistente" / "de Laboratório". O `FECHA_A_SECAO` entra como fronteira sem
    # ser origem: ele delimita o último verbete.
    fronteiras: list[tuple[int, int, str | None]] = []
    for i, (pagina, texto) in enumerate(linhas):
        junto = texto if i + 1 >= len(linhas) else f'{texto} {linhas[i + 1][1]}'
        achado = next((por_chave[chave(b)] for b in (texto, junto)
                       if chave(b) in por_chave), None)
        if achado is not None:
            fronteiras.append((i, pagina, achado))
        elif texto == FECHA_A_SECAO:
            fronteiras.append((i, pagina, None))

    queixas = []
    if not any(slug is None for _i, _p, slug in fronteiras):
        queixas.append(f'{FECHA_A_SECAO!r} não foi achado — o último verbete não tem '
                       f'fronteira de fim, e a descrição do poder único dele engole a '
                       f'prosa que vem depois')
    verbetes, vistos = {}, {}
    for n, (i, pagina, slug) in enumerate(fronteiras):
        if slug is None:
            continue
        fim = fronteiras[n + 1][0] if n + 1 < len(fronteiras) else len(linhas)
        corpo = [t for _p, t in linhas[i:fim]]
        if 'Benefícios.' not in junta(corpo):
            continue  # é menção na prosa, não verbete
        vistos[slug] = vistos.get(slug, 0) + 1
        if slug in verbetes:
            queixas.append(f'{nomes[slug]}: {vistos[slug]} trechos com o título e '
                           f'`Benefícios.` — o primeiro foi usado')
            continue
        verbetes[slug] = (pagina - OFFSET_DO_PDF, corpo)
    return verbetes, queixas


def le_lista(corpo: str) -> tuple[list[str], list[str], str | None] | None:
    """`Benefícios. A, B (perícias); C, D (poderes).` → as duas listas e a categoria
    do poder a sua escolha."""
    m = RE_LISTA.search(corpo)
    if not m:
        return None
    pericias = [p.strip() for p in m.group(1).split(',') if p.strip()]
    poderes, categoria = [], None
    for bruto in m.group(2).split(','):
        item = bruto.strip()
        if not item:
            continue
        if escolha := RE_ESCOLHA.search(item):
            categoria = escolha.group(1).lower()
            continue
        poderes.append(item)
    return pericias, poderes, categoria


def le_lista_da_tabela(bruto: str) -> tuple[list[str], list[str], str | None] | None:
    """A célula da tabela não rotula as duas metades: o `;` é que as separa."""
    if ';' not in bruto:
        return None
    return le_lista(f'Benefícios. {bruto.replace(";", " (perícias);")} (poderes).')


def le_o_poder_unico(corpo: list[str], poderes: list[str]) -> tuple[str, str] | None:
    """(título, descrição) do poder único, procurados LINHA a LINHA.

    O título é uma linha que É um dos poderes que a própria frase de benefícios
    lista — e a busca começa DEPOIS dessa frase, porque ela já os nomeia todos. A
    comparação é por `chave()`: o livro imprime o nome em versalete e o
    `pdftotext` devolve 'Quebra-galho' onde o catálogo escreve 'Quebra-Galho'.

    A descrição é cortada no ÚLTIMO ponto, que é o que descarta a legenda da
    ilustração — 'Aventureiros podem ter as origens mais surpreendentes' não
    termina em ponto, e ela fica entre a descrição do Esforçado e o fim da seção.
    """
    fecha = next((i for i, linha in enumerate(corpo) if '(poder' in linha), None)
    if fecha is None:
        return None
    por_chave = {chave(p): p for p in poderes}
    titulo = next(((i, por_chave[chave(linha)])
                   for i, linha in enumerate(corpo[fecha + 1:], fecha + 1)
                   if chave(linha) in por_chave), None)
    if titulo is None:
        return None
    i, nome = titulo
    descricao = junta(corpo[i + 1:])
    return nome, descricao[:descricao.rfind('.') + 1]


def numeros(texto: str) -> set[str]:
    """Os números que uma frase imprime, com o sinal normalizado. O travessão do
    livro e o hífen do catálogo são o mesmo menos."""
    return {n.translate({ord(c): '-' for c in '–−'}).lstrip('+')
            for n in RE_NUMERO.findall(RE_REMISSAO.sub(' ', texto))}


def pericias_citadas(texto: str, nomes: list[str]) -> set[str]:
    return {n for n in nomes if re.search(rf'\b{re.escape(n)}\b', texto)}


def palavras_de_conteudo(texto: str) -> set[str]:
    """As palavras que carregam a regra, sem as vazias e sem acento."""
    sem_acento = ''.join(
        c for c in unicodedata.normalize('NFD', texto.lower())
        if unicodedata.category(c) != 'Mn')
    return {p for p in re.findall(r'[a-z0-9]+', sem_acento)
            if len(p) > 2 and p not in PALAVRAS_VAZIAS}


def cobertura(do_livro: str, do_catalogo: str) -> tuple[float, int, int]:
    """Quanto das palavras de conteúdo do LIVRO a frase do catálogo repete.

    A direção importa: o denominador é o LIVRO, porque a pergunta é "a regra do
    livro está aqui?" e não "o catálogo é conciso?". Uma frase mais curta que
    preserve a regra pontua alto; uma frase longa que fale de outra coisa pontua
    zero por mais bem escrita que seja.
    """
    livro = palavras_de_conteudo(do_livro)
    if not livro:
        return 1.0, 0, 0
    juntas = livro & palavras_de_conteudo(do_catalogo)
    return len(juntas) / len(livro), len(juntas), len(livro)


def pericia_base(nome: str) -> str:
    """`Ofício (alquimista)` → `Ofício`: a perícia que o motor treina."""
    return RE_ESPECIALIZACAO.sub('', nome)


def confere_os_dois_catalogos(fonte: dict, achatado: dict) -> list[str]:
    """Os dois arquivos descrevem as MESMAS 35 origens e compartilham o `uid`.

    Uma divergência aqui é defeito independente do livro, e é a mais barata de
    achar: não precisa de PDF nenhum. O que se confere é o que os dois dizem
    sobre a mesma coisa — nome, uid, poder único, e a lista de benefícios do
    achatado contra `pericias + poderes − poderUnico` da fonte.
    """
    fora = []
    for slug in sorted(set(fonte) | set(achatado)):
        if slug not in achatado:
            fora.append(f'{slug}: está em origins-source.json e não em origins.json')
            continue
        if slug not in fonte:
            fora.append(f'{slug}: está em origins.json e não em origins-source.json')
            continue
        f, a = fonte[slug], achatado[slug]
        if f['name'] != a['name']:
            fora.append(f'{slug}: nome {f["name"]!r} na fonte × {a["name"]!r} no achatado')
        if f['uid'] != a['uid']:
            fora.append(f'{slug}: uid {f["uid"]} na fonte × {a["uid"]} no achatado — '
                        f'o banco aponta para um deles')
        unico = a.get('poderUnico', {}).get('name')
        if f.get('poderUnico') != unico:
            fora.append(f'{slug}: poder único {f.get("poderUnico")!r} na fonte × '
                        f'{unico!r} no achatado')
        esperados = [pericia_base(p) for p in f['pericias']] + [
            p for p in f['poderes'] if p != f.get('poderUnico')]
        # O benefício de escolha ("Poder da Tormenta (escolha)") não tem nome de
        # poder na fonte: ela o guarda como `poderChoiceCategory`, e o achatado o
        # monta como um benefício com `powerPick`.
        achados = [b['name'] for b in a['benefits'] if not b.get('powerPick')]
        escolhas = [b['powerPick'] for b in a['benefits'] if b.get('powerPick')]
        if sorted(chave(n) for n in esperados) != sorted(chave(n) for n in achados):
            fora.append(f'{slug}: benefícios {achados} no achatado × '
                        f'{esperados} derivados da fonte')
        categoria = f.get('poderChoiceCategory')
        if sorted(escolhas) != sorted([categoria] if categoria else []):
            fora.append(f'{slug}: escolha de poder {escolhas} no achatado × '
                        f'{[categoria] if categoria else []} na fonte')
    return fora


def confere_a_lista(origem: dict, lido, medidos: dict) -> list[str]:
    """As duas listas de benefícios e a categoria do poder a sua escolha."""
    pericias, poderes, categoria = lido
    fora = []
    medidos['pericias'] += len(pericias)
    if [chave(p) for p in pericias] != [chave(p) for p in origem['pericias']]:
        fora.append(f'perícias: catálogo {origem["pericias"]} × livro {pericias}')
    medidos['poderes'] += len(poderes)
    if [chave(p) for p in poderes] != [chave(p) for p in origem['poderes']]:
        fora.append(f'poderes: catálogo {origem["poderes"]} × livro {poderes}')
    if categoria != origem.get('poderChoiceCategory'):
        fora.append(f'poderChoiceCategory: catálogo '
                    f'{origem.get("poderChoiceCategory")!r} × livro {categoria!r}')
    return fora


def confere_a_regra_do_unico(do_livro: str, unico: str, no_achatado: dict | None,
                             nomes_de_pericia: list[str], medidos: dict) -> list[str]:
    """Os NÚMEROS e as PERÍCIAS que a descrição do poder único imprime.

    A prosa não se compara — o catálogo encurta de propósito. Estas duas se
    comparam porque uma forma mais curta as PRESERVA: uma descrição que perdeu o
    número do livro ou que nomeia outra perícia não é uma reescrita, é outra
    regra. O veredito sobre a frase é humano, e é por isso que as duas saem
    impressas lado a lado.
    """
    if no_achatado is None:
        return [f'poder único {unico!r}: origins.json não traz a descrição dele']
    if not do_livro:
        return [f'poder único {unico!r}: o verbete não traz descrição depois do título']
    do_catalogo = no_achatado.get('description', '')
    medidos['regra_do_unico'] += 1
    fora = []
    if numeros(do_livro) != numeros(do_catalogo):
        fora.append(f'{unico}: números {sorted(numeros(do_catalogo))} no catálogo × '
                    f'{sorted(numeros(do_livro))} no livro')
    do_livro_p = pericias_citadas(do_livro, nomes_de_pericia)
    do_catalogo_p = pericias_citadas(do_catalogo, nomes_de_pericia)
    if do_livro_p != do_catalogo_p:
        fora.append(f'{unico}: perícias citadas {sorted(do_catalogo_p)} no catálogo × '
                    f'{sorted(do_livro_p)} no livro')
    razao, juntas, total = cobertura(do_livro, do_catalogo)
    medidos['cobertura'].append((razao, unico))
    if razao < COBERTURA_MINIMA:
        fora.append(f'{unico}: o catálogo repete {juntas} das {total} palavras de '
                    f'conteúdo da regra do livro ({razao:.0%}, piso '
                    f'{COBERTURA_MINIMA:.0%}) — não é uma forma mais curta, é outra '
                    f'regra')
    if fora:
        fora.append(f'  {unico} NO LIVRO:    {do_livro!r}')
        fora.append(f'  {unico} NO CATÁLOGO: {do_catalogo!r}')
    return fora


def confere_os_itens(origem: dict, corpo: str, medidos: dict) -> list[str]:
    """A linha `Itens.` do verbete contra `itensIniciais`.

    O catálogo reescreve a frase do livro em itens curtos ("Joia de família
    (T$ 300)" para "Joia de família no valor de T$ 300"), então o que se compara é
    o SUBSTANTIVO de cada item: a primeira palavra significativa do item do
    catálogo tem de aparecer na frase do livro. É o que pega item inventado e item
    perdido sem opinar sobre a redação.
    """
    m = RE_ITENS.search(corpo)
    if not m:
        return ['itens: o verbete não imprime uma linha `Itens.` fechada em ponto']
    do_livro, fora = chave(m.group(1)), []
    for item in origem['itensIniciais']:
        palavras = [p for p in re.findall(r'\w+', item) if len(p) > 3]
        if not palavras:
            continue
        medidos['itens'] += 1
        if chave(palavras[0]) not in do_livro:
            fora.append(f'itens: {item!r} — {palavras[0]!r} não aparece na linha '
                        f'`Itens.` do livro: {m.group(1)!r}')
    return fora


def confere(slug: str, origem: dict, da_tabela, do_verbete, corpo: list[str],
            pagina: int | None, nomes_de_pericia: list[str], no_achatado: dict | None,
            medidos: dict) -> tuple[list[str], list[str]]:
    """As queixas desta origem, e as linhas em que o LIVRO discorda de si mesmo."""
    fora, livro = [], []
    declarada = slug in DISCORDANCIA_DECLARADA
    briga = bool(da_tabela and do_verbete and da_tabela != do_verbete)
    if briga and not declarada:
        livro.append(f'{origem["name"]}: a Tabela 1-19 diz {da_tabela} e o verbete '
                     f'diz {do_verbete} — discordância NOVA, não declarada')
    if declarada and not briga:
        livro.append(f'{origem["name"]}: a discordância declarada '
                     f'({DISCORDANCIA_DECLARADA[slug]}) NÃO acontece mais — tire a '
                     f'linha de DISCORDANCIA_DECLARADA')
    junto = junta(corpo)
    # Onde o livro discorda de si, o catálogo segue a declaração que a tabela de
    # DISCORDANCIA_DECLARADA nomeia — aqui, a da Tabela 1-19.
    lido = da_tabela if declarada else (do_verbete or da_tabela)
    if lido is not None:
        fora += confere_a_lista(origem, lido, medidos)
        achado = le_o_poder_unico(corpo, lido[1])
        if achado is None:
            fora.append('poder único: o verbete não imprime, depois da frase de '
                        'benefícios, nenhum título que seja um dos poderes listados')
        else:
            medidos['unico'] += 1
            unico, descricao = achado
            if chave(unico) != chave(origem.get('poderUnico') or ''):
                fora.append(f'poderUnico: catálogo {origem.get("poderUnico")!r} × '
                            f'livro {unico!r}')
            fora += confere_a_regra_do_unico(descricao, unico, no_achatado,
                                            nomes_de_pericia, medidos)
    if corpo:
        fora += confere_os_itens(origem, junto, medidos)
    if pagina is not None:
        medidos['pagina'] += 1
        if origem['bookPage'] != pagina:
            fora.append(f'bookPage: catálogo {origem["bookPage"]} × livro {pagina} '
                        f'(onde o TÍTULO do verbete é impresso)')
    return fora, livro


def main() -> int:
    fonte = json.load(open(FONTE, encoding='utf-8'))
    achatado = {o['id']: o for o in json.load(open(ACHATADO, encoding='utf-8'))}
    nomes_de_pericia = [p['name'] for p in json.load(open(PERICIAS, encoding='utf-8'))]
    nomes = {slug: o['name'] for slug, o in fonte.items()}

    entre_catalogos = confere_os_dois_catalogos(fonte, achatado)
    da_tabela, notas, consumidas = le_tabela(nomes)
    verbetes, queixas = le_verbetes(nomes, set(consumidas))

    medidos: dict = dict.fromkeys(
        ('pericias', 'poderes', 'unico', 'regra_do_unico', 'pagina', 'itens'), 0)
    medidos['cobertura'] = []
    divergem, nas_duas, numa_so, em_nenhuma, do_livro = 0, 0, 0, [], []
    for slug, origem in fonte.items():
        pagina, corpo = verbetes.get(slug, (None, []))
        tabela = le_lista_da_tabela(da_tabela[slug]) if slug in da_tabela else None
        verbete = le_lista(junta(corpo)) if corpo else None
        if slug == SEM_LISTA:
            queixas.append(
                f'{origem["name"]}: a EXCEÇÃO NOMEADA — o livro não imprime '
                f'"(perícias); … (poderes)." para ela nas duas declarações, porque os '
                f'benefícios dela são escolhidos pelo mestre. Dela se medem a página e '
                f'os itens; a lista e a regra do poder único se conferem no OLHO')
        elif tabela is None and verbete is None:
            em_nenhuma.append(f'{origem["name"]}: não ancorou em nenhuma das duas '
                              f'declarações — a lista de benefícios NÃO foi medida')
        elif tabela is None or verbete is None:
            numa_so += 1
            faltou = 'Tabela 1-19' if tabela is None else 'verbete'
            queixas.append(f'{origem["name"]}: não ancorou no {faltou} — a lista foi '
                           f'lida numa declaração só, sem se conferir contra a outra')
        else:
            nas_duas += 1
        if slug == SEM_LISTA:
            tabela = verbete = None
        fora, discorda = confere(slug, origem, tabela, verbete, corpo, pagina,
                                 nomes_de_pericia,
                                 achatado.get(slug, {}).get('poderUnico'), medidos)
        do_livro += discorda
        for queixa in fora:
            print(f'  {origem["name"]}: {queixa}' if not queixa.startswith('  ')
                  else f'  {queixa}')
            divergem += not queixa.startswith('  ')

    if do_livro:
        print('\nO LIVRO DISCORDA DE SI MESMO (achado sobre o LIVRO, não sobre o '
              'catálogo):')
        for linha in do_livro:
            print(f'  {linha}')
    for slug, motivo in DISCORDANCIA_DECLARADA.items():
        print(f'  DISCORDÂNCIA DECLARADA {fonte[slug]["name"]}: {motivo}')
    if entre_catalogos:
        print('\nOS DOIS CATÁLOGOS DISCORDAM (defeito sem precisar do livro):')
        for linha in entre_catalogos:
            print(f'  {linha}')
    if em_nenhuma or queixas:
        print()
        for linha in em_nenhuma + queixas:
            print(f'  NÃO MEDIDO {linha}')

    print('\na Tabela 1-19, medida:')
    for nota in notas:
        print(f'  {nota}')
    print(f'\norigens: {len(fonte)} | ancoraram nas DUAS declarações: {nas_duas} '
          f'| em UMA só: {numa_so} | em NENHUMA (≠ corretas): {len(em_nenhuma)} '
          f'| exceção nomeada: {SEM_LISTA}')
    print(f'verbetes achados: {len(verbetes)}/{len(fonte)} | linhas da tabela: '
          f'{len(da_tabela)}/{len(fonte)}')
    coberturas = medidos.pop('cobertura')
    print('BENEFÍCIOS COMPARADOS: ' + ', '.join(f'{c} {n}' for c, n in medidos.items()))
    # A cobertura de TODAS sai impressa, e não só a das reprovadas: uma lista de
    # reprovados vazia e um medidor que não mediu se parecem no terminal.
    print(f'cobertura da regra do poder único, da pior para a melhor '
          f'({len(coberturas)} medidas, piso {COBERTURA_MINIMA:.0%}):')
    for razao, nome in sorted(coberturas):
        print(f'  {razao:5.0%} {nome}')
    print(f'divergem do livro: {divergem} | o livro discorda de si: {len(do_livro)} '
          f'| os dois catálogos discordam: {len(entre_catalogos)}')
    return 1 if divergem or do_livro or entre_catalogos or em_nenhuma else 0


if __name__ == '__main__':
    raise SystemExit(main())
