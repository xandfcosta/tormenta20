import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { createSignal } from 'solid-js'
import { describe, expect, it, vi } from 'vitest'
import { Dialog, DialogContent, DialogDescription, DialogTitle, DialogTrigger } from './dialog'
import { Popover, PopoverContent, PopoverTrigger } from './popover'
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
        <TabsContent value="visao">Resumo da crônica</TabsContent>
        <TabsContent value="membros">Roster do grupo</TabsContent>
      </Tabs>
    )
  }

  it('mostra o painel da aba inicial', () => {
    render(() => <Fixture />)
    expect(screen.getByText('Resumo da crônica')).toBeInTheDocument()
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

  it('navega pelas abas com as setas', async () => {
    render(() => <Fixture />)
    const user = userEvent.setup()
    await user.click(screen.getByRole('tab', { name: 'Visão geral' }))
    await user.keyboard('{ArrowRight}')
    await waitFor(() => expect(screen.getByRole('tab', { name: 'Membros' })).toHaveFocus())
  })
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

  it('fecha no Escape', async () => {
    render(() => <Fixture />)
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Abrir' }))
    await screen.findByRole('dialog')
    await user.keyboard('{Escape}')
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

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

describe('Popover', () => {
  it('abre e fecha pelo gatilho', async () => {
    render(() => (
      <Popover>
        <PopoverTrigger>Filtros</PopoverTrigger>
        <PopoverContent>Só mestrando</PopoverContent>
      </Popover>
    ))
    const user = userEvent.setup()
    expect(screen.queryByText('Só mestrando')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Filtros' }))
    expect(await screen.findByText('Só mestrando')).toBeInTheDocument()

    await user.keyboard('{Escape}')
    await waitFor(() => expect(screen.queryByText('Só mestrando')).not.toBeInTheDocument())
  })
})

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
