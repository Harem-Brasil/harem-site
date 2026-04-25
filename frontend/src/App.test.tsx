import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import App from './App'

describe('App', () => {
  it('renders the Harem Brasil heading', () => {
    render(<App />)
    expect(screen.getByRole('heading', { name: /Harem Brasil/i })).toBeInTheDocument()
  })

  it('renders the age restriction tag', () => {
    render(<App />)
    expect(screen.getByText(/EXCLUSIVO PARA MAIORES DE 18 ANOS/i)).toBeInTheDocument()
  })

  it('renders the CTA button linking to WhatsApp', () => {
    render(<App />)
    const link = screen.getByRole('link', { name: /Quero entrar primeiro/i })
    expect(link).toBeInTheDocument()
    expect(link).toHaveAttribute('href', expect.stringContaining('wa.me'))
  })

  it('renders the footer text', () => {
    render(<App />)
    expect(screen.getByText(/Plataforma em desenvolvimento/i)).toBeInTheDocument()
  })
})
