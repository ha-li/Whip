import React from 'react'
import ReactDOM from 'react-dom/client'
// Global styles first: component sheets (imported by the components) override them.
import './styles/base.css'
import './styles/controls.css'
import App from './App'

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
