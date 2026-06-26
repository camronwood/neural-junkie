import './monacoSetup';
import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import './styles.css';
import { initContextMenuPolicy } from './utils/contextMenuPolicy';

initContextMenuPolicy();

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);

