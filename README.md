# auth-api

API de autenticação segura em Go: registo, login, JWT com refresh token e papéis (`user` e `admin`). Primeiro construída, depois atacada e protegida.

> Estado: em desenvolvimento (projeto de portefólio).

## Objetivo

Construir um serviço de identidade completo e depois protegê-lo, documentando cada decisão de segurança. A história do projeto é "construí e depois protegi".

## Stack

- Go (net/http ou chi)
- PostgreSQL (pgx e golang-migrate), em Docker
- argon2id para as palavras-passe
- JWT (golang-jwt/jwt v5)
- GitHub Actions, Dependabot, Trivy e OWASP ZAP

## Funcionalidades previstas

- Registo e login com palavras-passe guardadas com argon2id
- JWT de acesso com expiração curta (15 min) e refresh token com rotação
- Papéis `user` e `admin`, com rotas protegidas por middleware
- Rate limiting no login e mensagens de erro genéricas

## Rotas previstas

- GET /healthz (público)
- POST /auth/register (público)
- POST /auth/login (público)
- POST /auth/refresh (refresh token)
- POST /auth/logout (autenticado)
- GET /me (user ou admin)
- GET /admin/users (só admin)
- PATCH /admin/users/{id}/role (só admin)

## Plano

### Parte 1: Construir

1. Módulo Go e endpoint /healthz
2. Configuração por variáveis de ambiente e docker-compose com PostgreSQL
3. Migrações das tabelas users e refresh_tokens
4. Registo com argon2id e validação de input
5. Login com JWT de acesso
6. Refresh token com rotação e logout
7. Middleware de autenticação e de papéis
8. Testes unitários e de integração
9. Rate limiting no login

### Parte 2: Proteger

10. Dockerfile multi-stage com imagem distroless e utilizador non-root
11. Dependabot (gomod, docker e github-actions)
12. Pipeline no GitHub Actions: testes, build da imagem e Trivy a falhar em vulnerabilidades HIGH e CRITICAL
13. Relatório inicial do OWASP ZAP
14. Correções, um commit por problema, e relatório final do ZAP
15. Diagrama do fluxo de login e secção "Decisões de segurança"

## Decisões de segurança

A preencher à medida que o projeto avança (porquê argon2id, porquê rotação de refresh tokens, porquê distroless, etc.).

## Como correr

A preencher quando existir o docker-compose.

## Licença

MIT
