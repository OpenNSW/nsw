import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { Theme } from '@radix-ui/themes'
import { AsgardeoProvider } from '@asgardeo/react'
import "@radix-ui/themes/styles.css"
import './index.css'
import App from './App.tsx'

const getRequiredEnv = (name: keyof ImportMetaEnv): string => {
  const value = import.meta.env[name]
  if (!value || value.trim() === '') {
    throw new Error(`Missing required environment variable: ${name}`)
  }

  return value
}

const APP_URL = import.meta.env.VITE_APP_URL || window.location.origin
const CLIENT_ID = getRequiredEnv('VITE_IDP_CLIENT_ID')
const IDP_BASE_URL = getRequiredEnv('VITE_IDP_BASE_URL')
const IDP_PLATFORM = import.meta.env.VITE_IDP_PLATFORM || 'AsgardeoV2'
const IDP_SCOPES = import.meta.env.VITE_IDP_SCOPES
  ? import.meta.env.VITE_IDP_SCOPES.split(',').map((scope: string) => scope.trim())
  : ['openid', 'profile', 'email']

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AsgardeoProvider
      clientId={CLIENT_ID}
      baseUrl={IDP_BASE_URL}
      platform={IDP_PLATFORM}
      afterSignInUrl={APP_URL}
      afterSignOutUrl={APP_URL}
      scopes={IDP_SCOPES}
    >
      <Theme scaling="110%">
        <BrowserRouter>
          <App />
        </BrowserRouter>
      </Theme>
    </AsgardeoProvider>
  </StrictMode>,
)
