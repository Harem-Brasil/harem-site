# Front-end React + Vite (tema discreto)

Base inicial migrada para React com Vite, com visual escuro/preto e layout discreto.

## Estrutura

- `index.html`: entrada da aplicacao.
- `src/main.jsx`: bootstrap do React.
- `src/App.jsx`: layout principal e interacoes.
- `src/mockData.js`: dados mockados de feed e forum.
- `src/styles.css`: tema e responsividade.
- `vite.config.js`: configuracao do Vite.
- `package.json`: dependencias e scripts.

## Funcionalidades iniciais

- Navegacao entre secoes (`Inicio`, `Forum`, `Mensagens`, `Assinaturas`, `Perfil`).
- Composer para novo post/topico.
- Feed principal com filtros quando a aba `Forum` estiver ativa.
- Painel lateral com salas de forum e topicos em alta.

## Como rodar

1. Instalar Node.js (se ainda nao estiver instalado).
2. No diretorio `frontend`, rodar:
   - `npm install`
   - `npm run dev`
3. Abrir a URL mostrada no terminal (geralmente `http://localhost:5173`).

## Proximos passos sugeridos

- Separar o layout em componentes (`Sidebar`, `Feed`, `ForumPanel`).
- Adicionar React Router para rotas reais.
- Integrar com API do backend (autenticacao, posts, forum).
