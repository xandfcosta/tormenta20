import { createFileRoute } from '@tanstack/solid-router'
import { HomePage } from '@/pages/home/home-page'
import { requireSession } from './-guards'

// `/` is the Hub — the game's main menu. Anonymous visitors have no landing
// page; they go straight to login.
export const Route = createFileRoute('/')({
  beforeLoad: requireSession,
  component: HomePage,
})
