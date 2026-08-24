import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { createSignal } from 'solid-js'
import { describe, expect, it, vi } from 'vitest'
import { Dialog, DialogContent, DialogDescription, DialogTitle, DialogTrigger } from './dialog'
import { type SelectOption, Select } from './select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from './tabs'

describe('Tabs', () => {
  function Fixture() {
    return (
      <Tabs defaultValue="visao">
        <TabsList>
          <TabsTrigger value="visao">Visão geral</TabsTrigger>
          <TabsTrigger value="membros">Membros</TabsTrigger>
        </TabsList>
        <TabsContent value="visao">Resumo da campanha</TabsContent>
        <TabsContent value="membros">Roster do grupo</TabsContent>
      </Tabs>
    )
  }

  it('mostra o painel da aba inicial', () => {
    render(() => <Fixture />)
    expect(screen.getByText('Resumo da campanha')).toBeInTheDocument()
  })

  it('troca de painel no clique', async () => {
    render(() => <Fixture />)
    await userEvent.setup().click(screen.getByRole('tab', { name: 'Membros' }))
    expect(await screen.findByText('Roster do grupo')).toBeInTheDocument()
  })

  // Kobalte marks the active trigger with `data-selected`, where Radix used
  // `data-state="active"` — the classes in tabs.tsx depend on this.
  it('marca a aba ativa com data-selected (contrato das classes)', async () => {
    render(() => <Fixture />)
    await userEvent.setup().click(screen.getByRole('tab', { name: 'Membros' }))
    await waitFor(() =>
      expect(screen.getByRole('tab', { name: 'Membros' })).toHaveAttribute('data-selected'),
    )
    expect(screen.getByRole('tab', { name: 'Visão geral' })).not.toHaveAttribute('data-selected')
  })

  // 'navega pelas abas com as setas' saiu na ALE-187: é o roving tabindex do
  // Kobalte, testado pelo Kobalte. O que ESTA casa decide sobre as abas — o
  // `data-selected` de que as classes dependem — fica logo acima.
})

describe('Dialog', () => {
  function Fixture() {
    return (
      <Dialog>
        <DialogTrigger>Abrir</DialogTrigger>
        <DialogContent>
          <DialogTitle>Excluir personagem</DialogTitle>
          <DialogDescription>Essa ação não pode ser desfeita.</DialogDescription>
        </DialogContent>
      </Dialog>
    )
  }

  it('começa fechado', () => {
    render(() => <Fixture />)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('abre pelo gatilho, com título e descrição ligados', async () => {
    render(() => <Fixture />)
    await userEvent.setup().click(screen.getByRole('button', { name: 'Abrir' }))
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveAccessibleName('Excluir personagem')
    expect(dialog).toHaveAccessibleDescription('Essa ação não pode ser desfeita.')
  })

  // 'fecha no Escape' saiu na ALE-187: é o `Dialog` do Kobalte fazendo o que
  // ele documenta. O caso abaixo fica, e não é o mesmo: ele prende o rótulo
  // pt-BR do botão de fechar, que a biblioteca entrega em INGLÊS
  // (`aria-label="Dismiss"`) e que a casa sobrescreve — armadilha real, e
  // nossa.

  // Kobalte's CloseButton ships aria-label="Dismiss"; dialog.tsx overrides it
  // so a pt-BR app doesn't announce English.
  it('fecha no botão de fechar', async () => {
    render(() => <Fixture />)
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Abrir' }))
    await screen.findByRole('dialog')
    await user.click(screen.getByRole('button', { name: 'Fechar' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('showCloseButton={false} esconde o X', async () => {
    render(() => (
      <Dialog>
        <DialogTrigger>Abrir</DialogTrigger>
        <DialogContent showCloseButton={false}>
          <DialogTitle>Sem X</DialogTitle>
        </DialogContent>
      </Dialog>
    ))
    await userEvent.setup().click(screen.getByRole('button', { name: 'Abrir' }))
    await screen.findByRole('dialog')
    expect(screen.queryByRole('button', { name: 'Fechar' })).not.toBeInTheDocument()
  })
})

// O bloco `Popover` saiu inteiro na ALE-187: o único caso dele afirmava que o
// gatilho abre e fecha, que é o Popover do Kobalte fazendo o que documenta.
// Nada desta casa passava por ali.

describe('Select', () => {
  const RACES: SelectOption<string>[] = [
    { value: 'humano', label: 'Humano' },
    { value: 'anao', label: 'Anão' },
    { value: 'goblin', label: 'Goblin', disabled: true },
  ]

  function Fixture(props: { onChange?: (option: SelectOption<string> | null) => void }) {
    const [value, setValue] = createSignal<SelectOption<string> | null>(null)
    return (
      <Select
        aria-label="Raça"
        options={RACES}
        value={value()}
        placeholder="Escolha"
        onChange={(option) => {
          setValue(option)
          props.onChange?.(option)
        }}
      />
    )
  }

  it('mostra o placeholder enquanto nada foi escolhido', () => {
    render(() => <Fixture />)
    expect(screen.getByText('Escolha')).toBeInTheDocument()
  })

  // Kobalte composes the trigger's accessible name from the label AND the
  // current value ("Raça Escolha"), so match by prefix rather than exactly.
  it('escolhe uma opção e devolve o valor pro chamador', async () => {
    const onChange = vi.fn()
    render(() => <Fixture onChange={onChange} />)
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: /^Raça/ }))
    await user.click(await screen.findByRole('option', { name: 'Anão' }))

    expect(onChange).toHaveBeenCalledWith({ value: 'anao', label: 'Anão' })
    expect(await screen.findByText('Anão')).toBeInTheDocument()
  })

  it('respeita opções desabilitadas', async () => {
    render(() => <Fixture />)
    await userEvent.setup().click(screen.getByRole('button', { name: /^Raça/ }))
    expect(await screen.findByRole('option', { name: 'Goblin' })).toHaveAttribute('data-disabled')
  })
})
