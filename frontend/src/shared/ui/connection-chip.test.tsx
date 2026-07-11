import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ConnectionChip } from './connection-chip'

describe('ConnectionChip', () => {
  it('shows "Conectado" when status=connected', () => {
    render(<ConnectionChip status="connected" />)
    expect(screen.getByText('Conectado')).toBeInTheDocument()
  })

  it('shows "Reconectando…" when status=reconnecting', () => {
    render(<ConnectionChip status="reconnecting" />)
    expect(screen.getByText('Reconectando…')).toBeInTheDocument()
  })

  it('shows "Offline" when status=offline', () => {
    render(<ConnectionChip status="offline" />)
    expect(screen.getByText('Offline')).toBeInTheDocument()
  })

  it('overrides the label with an unsaved-changes warning when dirty', () => {
    render(<ConnectionChip status="connected" dirty />)
    expect(screen.getByText('Alterações não salvas')).toBeInTheDocument()
    expect(screen.queryByText('Conectado')).not.toBeInTheDocument()
  })

  it('drops the text in compact mode but keeps the aria-label', () => {
    render(<ConnectionChip status="offline" compact />)
    expect(screen.queryByText('Offline')).not.toBeInTheDocument()
    expect(screen.getByRole('status', { name: 'Offline' })).toBeInTheDocument()
  })
})
