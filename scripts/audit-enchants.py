#!/usr/bin/env python3
"""Confere os ENCANTOS DE ARMA do catálogo contra o livro (ALE-411).

As duas âncoras, e o que elas conferem uma na outra
---------------------------------------------------
O livro declara os encantos DUAS vezes, e as duas leituras se cobram antes de
qualquer correção no catálogo:

* a **Tabela 8-8** (p336), `d% | Encanto | Efeito`, que é a faixa de rolagem e o
  resumo de uma linha;
* o **verbete** (p335-336), na forma `Nome. Regra`, que é a regra inteira.

Um nome que aparece numa e não na outra é defeito de LEITURA, e ele sai antes de
o catálogo ser mencionado — foi assim que o `audit-origins.py` descobriu que a
página saía fora de ordem.

O denominador é de COBERTURA, e a diferença importa
---------------------------------------------------
Nos auditores da ALE-391 havia duas coisas para comparar: o livro e um catálogo
que já existia. Aqui o catálogo COMEÇA vazio, então "transcrevi os encantos" e
"transcrevi doze dos vinte e oito" têm a mesma cara no terminal. Por isso o
relatório afirma, sempre: quantos a tabela tem, quantos entraram, e quantos
ficaram sem modificador — com o MOTIVO de cada um.

Um encanto com modificador não está pronto por isso
---------------------------------------------------
Quase todo encanto tem duas metades: o Flamejante é "+1d6 de fogo" E "gaste 2 PM
para disparar uma bola de fogo". Medir só "tem modifiers?" daria verde a quem
modelou 30% da regra. A terceira dimensão é a COBERTURA de palavras do verbete
pela descrição do catálogo, que é o instrumento que a ALE-405 teve de inventar
quando os números coincidiram por acaso.
"""
import json
import pathlib
import re
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import t20pdf as P

# A Tabela 8-8 mora na p336 e os verbetes atravessam as duas páginas.
# O livro impresso está 6 páginas atrás do PDF, e a conta não é repetida aqui de
# propósito: ela é do `engine-go/CLAUDE.md`, que é o dono dela.
OFFSET_DO_PDF = 6
PAGINA_DA_TABELA = 342
PAGINAS_DOS_VERBETES = (341, 342)

# As colunas da Tabela 8-8, medidas. A do dado é uma FAIXA e não um x, porque a
# célula é alinhada à DIREITA: "01-05" começa em 59,1 e "46" em 65,1.
#
# E ela é lida SEM o `sem_lixo`, que é a armadilha desta página: o padrão de
# número de página (`^\d{1,3}$`) casa com as faixas de UM valor, e a limpeza
# comeu a Energética (46) e a Lancinante (64). Saíram 26 linhas com os nomes
# certos, as faixas certas e dois efeitos colados no vizinho de cima — o que
# denunciou foi a soma das faixas não fechar em 100. O número de página desta
# página mora em x=14,5, fora da janela.
X_DO_DADO = (56.0, 67.0)
X_DO_NOME = 90.8
X_DO_EFEITO = 154.3
FOLGA = 3.0

# A ÚLTIMA LINHA da tabela vem COLAPSADA num `<line>` só — "91-100 Arma
# específica Veja a Tabela 8-9" —, e é por isso que o padrão aceita cauda.
RE_FAIXA = re.compile(r'^(\d{1,3})(?:-(\d{1,3}))?(?:\s+(.*))?$')
# O verbete começa com o nome em NEGRITO seguido de ponto. O `[A-ZÁÉÍÓÚÂÊÔÃÕÇ]`
# na primeira letra é o que separa "Flamejante. A arma…" de uma frase qualquer
# que termine em nome próprio.
RE_VERBETE = re.compile(r'^([A-ZÁÉÍÓÚÂÊÔÃÕÇ][a-záéíóúâêôãõç]+)\. (.+)$')

# A LINHA QUE NÃO É ENCANTO. A Tabela 8-8 tem 29 linhas de dado e 28 encantos:
# "91-100 Arma específica Veja a Tabela 8-9" manda rolar em OUTRA tabela, e o
# próprio livro diz que ela apaga os encantos rolados. Declarada aqui para o
# denominador não subir para 29 em silêncio nem descer para 28 sem motivo.
NAO_E_ENCANTO = 'Arma específica'

# O ASTERISCO da tabela: "*Conta como dois encantos. Para itens menores, role
# novamente." Três o carregam, e ele é regra — o catálogo o guarda em `counts`.
CONTAM_DOBRADO = {'Energética', 'Lancinante', 'Magnífica'}

# ONDE OS VERBETES ACABAM. O último (Venenosa) não tem sucessor que o pare, e
# sem esta linha ele engolia a seção seguinte inteira — "Armas Específicas" e o
# "Arco do Poder" viravam parte da regra dele, inflando a cobertura dele com
# palavras de outro assunto.
FIM_DOS_VERBETES = 'Armas Específicas'

# Os x da PROSA em cada página, medidos — o corpo e a primeira linha indentada
# de cada coluna. O que cai fora deles NÃO é verbete: na p336 a Tabela 8-8
# divide a página com a prosa (x de 51 a 154) e na p335 há um título de seção em
# x=458.
#
# Sem este recorte, a tabela inteira entrava como corpo da Magnífica e o título
# como corpo da Flamejante: 939 caracteres num verbete de 370, com o nome certo
# e a primeira frase certa. O que denunciou foi o TAMANHO, e por isso o relatório
# imprime o maior verbete — um medidor de prosa que não diz onde a prosa acaba
# infla a cobertura com palavras de outro assunto.
# Cada par é (corpo, primeira linha indentada), e o degrau é de 17 pontos.
COLUNAS_DA_PROSA = {
    341: (71.0, 88.0, 309.0, 326.0),
    342: (51.0, 68.0, 289.0, 306.0),
}

# E NA p336 O X NÃO BASTA: a prosa da coluna esquerda começa em 51 e a célula
# de d% da tabela cai entre 57 e 68 — as duas se sobrepõem. O que as separa é a
# ALTURA: a tabela acaba em y=527 (a nota do asterisco) e a prosa de baixo
# começa em y=596. Recortar só por x perdia três verbetes, entre eles o da
# Magnífica, e o relatório saía com 25 de 28 sem dizer que tinha perdido.
REGIAO_DA_TABELA = {342: (200.0, 540.0)}


def linha_da_tabela(pagina: int):
    """As linhas da Tabela 8-8: (inicio, fim, nome, efeito).

    A linha se monta pelo Y — as três células são blocos DIFERENTES, e o que
    diz que "01-05", "Ameaçadora" e "Duplica margem de ameaça" são a mesma linha
    é a altura das três ser igual. Parear por índice erra na primeira célula que
    quebra em duas linhas, e a da Caçadora quebra.
    """
    celulas = P.linhas_com_coordenada(pagina)
    dados = []
    for x, y, texto in celulas:
        if not X_DO_DADO[0] <= x <= X_DO_DADO[1]:
            continue
        casa = RE_FAIXA.match(texto)
        if casa:
            dados.append((y, casa.groups()))
    nomes = [(y, t) for x, y, t in celulas if abs(x - X_DO_NOME) < FOLGA]
    efeitos = sorted((y, t) for x, y, t in celulas if abs(x - X_DO_EFEITO) < FOLGA)

    for y, (inicio, fim, cauda) in sorted(dados):
        # A linha colapsada traz nome e efeito na própria cauda, e os dois se
        # separam pela tabela ser a última: "Arma específica Veja a Tabela 8-9".
        if cauda:
            nome, _, efeito = cauda.partition(' Veja ')
            yield int(inicio), int(fim or inicio), nome.strip(), ('Veja ' + efeito).strip()
            continue
        nome = next((t for yn, t in nomes if abs(yn - y) < P.MESMA_LINHA), None)
        if nome is None:
            continue
        # O efeito vai da altura desta linha até a da próxima: a célula que
        # quebra em duas (a Caçadora) tem a continuação SEM d% ao lado.
        proxima = next((yd for yd, _ in sorted(dados) if yd > y + P.MESMA_LINHA), 1e9)
        corpo = [t for ye, t in efeitos if y - P.MESMA_LINHA < ye < proxima - P.MESMA_LINHA]
        yield int(inicio), int(fim or inicio), nome.rstrip('*'), P.junta(corpo)


def faixas_que_nao_fecham(linhas) -> str:
    """A SOMA antes das parcelas: as faixas de d% têm de ladrilhar 1..100.

    É o denominador de graça de toda decomposição, e é o único controle que
    pega uma linha SUMIDA: 26 linhas com nomes certos e efeitos plausíveis se
    parecem com 29 no terminal, e a única coisa que não fecha é o dado.
    """
    esperado = 1
    for inicio, fim, nome, _ in linhas:
        if inicio != esperado:
            return (f'a faixa de {nome} começa em {inicio} e a anterior terminou '
                    f'em {esperado - 1}: falta linha entre as duas')
        esperado = fim + 1
    if esperado != 101:
        return f'as faixas terminam em {esperado - 1} e o d% vai até 100'
    return ''


def verbetes(paginas):
    """Os verbetes `Nome. Regra` das páginas, com a regra inteira.

    Um verbete termina onde o seguinte começa. A prosa dos encantos divide a
    página com a tabela, então a leitura tem de ser por COLUNA — é o que o
    `P.leitura` entrega.
    """
    achados: dict[str, list[str]] = {}
    pagina_de: dict[str, int] = {}
    atual = None
    for pagina in paginas:
        prosa = COLUNAS_DA_PROSA[pagina]
        limite = REGIAO_DA_TABELA.get(pagina)
        for x, y, texto in P.leitura(pagina):
            if not any(abs(x - coluna) < FOLGA for coluna in prosa):
                continue
            if limite and x < limite[0] and y < limite[1]:
                continue
            if texto.startswith(FIM_DOS_VERBETES):
                atual = None
                continue
            casa = RE_VERBETE.match(texto)
            if casa and casa.group(1) not in achados:
                atual = casa.group(1)
                achados[atual] = [casa.group(2)]
                pagina_de[atual] = pagina - OFFSET_DO_PDF
            elif atual is not None:
                achados[atual].append(texto)
    return {nome: (P.junta(corpo), pagina_de[nome]) for nome, corpo in achados.items()}


# O PISO DA COBERTURA. A descrição do catálogo é TRANSCRITA do verbete, então
# ela nasce em 100% — e é por isso que o piso é alto. O que este número prende
# não é a transcrição de hoje: é o dia em que alguém "melhorar" a frase e ela
# passar a dizer outra coisa, que foi o defeito da ALE-408 em dois arquivos de
# Poder Concedido.
PISO_DA_COBERTURA = 0.95

CATALOGO = P.RAIZ / 'engine-go/domain/catalog/data/items.json'
CATEGORIA = 'weapon-enchant'


def do_catalogo() -> dict:
    itens = json.loads(CATALOGO.read_text())
    return {i['name']: i for i in itens if i.get('category') == CATEGORIA}


def main() -> int:
    tabela = list(linha_da_tabela(PAGINA_DA_TABELA))
    falhas = []

    # ── ÂNCORA 1 contra ÂNCORA 2, antes de o catálogo ser mencionado ──────
    buraco = faixas_que_nao_fecham(tabela)
    if buraco:
        print(f'A TABELA 8-8 NÃO FECHA: {buraco}')
        print('Nada do que vem abaixo vale — o auditor leu a página errado.')
        return 1

    da_tabela = {nome: (inicio, fim, efeito)
                 for inicio, fim, nome, efeito in tabela if nome != NAO_E_ENCANTO}
    dos_verbetes = verbetes(PAGINAS_DOS_VERBETES)

    print(f'AS DUAS ÂNCORAS  tabela: {len(tabela)} linhas (soma fecha em 100), '
          f'{len(da_tabela)} encantos + "{NAO_E_ENCANTO}"')
    print(f'                 verbetes: {len(dos_verbetes)}')
    so_na_tabela = sorted(set(da_tabela) - set(dos_verbetes))
    so_no_verbete = sorted(set(dos_verbetes) - set(da_tabela))
    if so_na_tabela or so_no_verbete:
        print(f'  AS DUAS DISCORDAM — só na tabela: {so_na_tabela}; '
              f'só no verbete: {so_no_verbete}')
        print('  É defeito de LEITURA, e ele vem antes de qualquer correção de catálogo.')
        return 1
    print(f'                 as duas dizem os MESMOS {len(da_tabela)} nomes')

    # ── O CATÁLOGO ────────────────────────────────────────────────────────
    catalogo = do_catalogo()
    print(f'\nCATÁLOGO         {len(catalogo)} entradas em `{CATEGORIA}`')

    com_modificador, sem_modificador, cobertura_minima = 0, [], (2.0, '')
    for nome, (inicio, fim, _efeito) in sorted(da_tabela.items()):
        entrada = catalogo.get(nome)
        if entrada is None:
            falhas.append(f'{nome}: está na Tabela 8-8 e não está no catálogo')
            continue
        regra, pagina = dos_verbetes[nome]

        quanto, juntas, total = P.cobertura(regra, entrada.get('description', ''))
        cobertura_minima = min(cobertura_minima, (quanto, nome))
        if quanto < PISO_DA_COBERTURA:
            falhas.append(
                f'{nome}: a descrição do catálogo repete {juntas} das {total} palavras '
                f'de conteúdo do verbete ({quanto:.0%}) — abaixo do piso de '
                f'{PISO_DA_COBERTURA:.0%}. Ela deixou de dizer a regra do livro.')

        if (entrada.get('rollMin'), entrada.get('rollMax')) != (inicio, fim):
            falhas.append(
                f'{nome}: o catálogo diz d% {entrada.get("rollMin")}-{entrada.get("rollMax")} '
                f'e a Tabela 8-8 diz {inicio}-{fim}')
        esperado = 2 if nome in CONTAM_DOBRADO else 1
        if entrada.get('countsAs') != esperado:
            falhas.append(
                f'{nome}: o catálogo conta como {entrada.get("countsAs")} e o livro diz '
                f'{esperado} (o asterisco da tabela é "conta como dois encantos")')
        if entrada.get('bookPage') != pagina:
            falhas.append(f'{nome}: o catálogo cita a p{entrada.get("bookPage")} e o '
                          f'verbete está na p{pagina}')

        # O DENOMINADOR do que não foi modelado. Ter modificador e ter motivo
        # são MUTUAMENTE EXCLUSIVOS: um encanto com os dois é um motivo que
        # envelheceu, e ele mente com cara de registro.
        tem_modelo = bool(entrada.get('modifiers'))
        motivo = entrada.get('unmodeled', '')
        if tem_modelo:
            com_modificador += 1
        else:
            sem_modificador.append((nome, motivo))
        if not tem_modelo and not motivo:
            falhas.append(f'{nome}: não tem modificador NEM diz por quê')
        if tem_modelo and motivo:
            falhas.append(
                f'{nome}: tem modificador E diz que não foi modelado ({motivo!r}) — '
                f'uma das duas coisas envelheceu')

    print(f'  com modificador: {com_modificador}')
    print(f'  SEM modificador: {len(sem_modificador)}, e cada um diz por quê:')
    for nome, motivo in sem_modificador:
        print(f'    {nome:14s} {motivo}')
    print(f'\n  cobertura mínima: {cobertura_minima[0]:.0%} ({cobertura_minima[1]})')

    print(f'\nFALHAS: {len(falhas)}   (medidos: {len(da_tabela)} encantos, '
          f'{len(tabela)} linhas de tabela, {len(dos_verbetes)} verbetes)')
    for f in falhas:
        print(f'  {f}')
    return 1 if falhas else 0


if __name__ == '__main__':
    sys.exit(main())
