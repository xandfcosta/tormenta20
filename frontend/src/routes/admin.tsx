import { createFileRoute } from '@tanstack/solid-router'
import { AdminPage } from '@/pages/admin/admin-page'
import { requireAdmin } from './-guards'

export const Route = createFileRoute('/admin')({
  beforeLoad: requireAdmin,
  component: AdminPage,
})
