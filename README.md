# Ground Guard — Backend API

API REST do **Ground Guard**, uma plataforma IoT para monitoramento e automação de jardins e plantas. Permite gerenciar dispositivos, preferências de irrigação, vinculação via QR Code e conteúdos de suporte ao usuário.

Desenvolvido como TCC e preparado para evolução comercial.

---

## Sobre o projeto

O Ground Guard conecta sensores e atuadores em campo (umidade do solo, temperatura, bomba de irrigação, etc.) a um backend centralizado. Este repositório contém a **API em Go** que:

- Autentica usuários e gerencia sessões com tokens PASETO
- Registra e vincula dispositivos IoT a contas de usuário
- Armazena preferências de irrigação e histórico de ações
- Expõe conteúdos públicos (FAQ, tutoriais, ajuda e documentos legais)
- Serve imagens de perfil e QR Codes gerados para pareamento de dispositivos

A documentação interativa da API está disponível via **Swagger UI**.

---

## Stack tecnológica

| Camada | Tecnologia |
|--------|------------|
| Linguagem | Go 1.25+ |
| Framework HTTP | [Gin](https://github.com/gin-gonic/gin) |
| Banco de dados | PostgreSQL 16 |
| Queries tipadas | [sqlc](https://sqlc.dev/) |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) |
| Autenticação | PASETO (tokens simétricos) |
| Configuração | [Viper](https://github.com/spf13/viper) |
| Documentação | [Swaggo](https://github.com/swaggo/swag) |
| Containerização | Docker + Docker Compose |

---

## Estrutura do projeto

```
.
├── cmd/server/          # Entry point da aplicação
├── db/
│   ├── migration/       # Migrations SQL (golang-migrate)
│   ├── query/           # Queries SQL para o sqlc
│   ├── sqlc/            # Código Go gerado pelo sqlc
│   └── mock/            # Mocks gerados (mockgen)
├── docs/                # Swagger gerado (swag)
├── internal/
│   ├── dto/             # Data Transfer Objects
│   ├── handler/         # Handlers HTTP
│   ├── middleware/      # Middlewares (auth, etc.)
│   └── routes/          # Registro de rotas
├── token/               # Geração e validação de tokens
├── util/                # Utilitários (config, senha, QR Code, etc.)
├── uploads/             # Arquivos estáticos (perfil, QR Codes)
├── docker-compose.yaml
├── Dockerfile
└── Makefile
```

---

## Pré-requisitos

Para rodar localmente (sem Docker):

- [Go](https://go.dev/dl/) 1.25 ou superior
- [Docker](https://www.docker.com/) (para o PostgreSQL)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html) (opcional, para regenerar queries)

Para rodar com Docker Compose, basta ter Docker e Docker Compose instalados.

---

## Configuração

Crie um arquivo `app.env` na raiz do projeto (o arquivo está no `.gitignore`):

```env
DB_DRIVER=postgres
DB_SOURCE=postgresql://root:secret@localhost:5433/ground_guard?sslmode=disable
SERVER_ADDRESS=0.0.0.0:8080
TOKEN_SYMMETRIC_KEY=12345678901234567890123456789012
ACCESS_TOKEN_DURATION=15m
REFRESH_TOKEN_DURATION=168h
```

| Variável | Descrição |
|----------|-----------|
| `DB_DRIVER` | Driver do banco (`postgres`) |
| `DB_SOURCE` | Connection string do PostgreSQL |
| `SERVER_ADDRESS` | Endereço e porta do servidor HTTP |
| `TOKEN_SYMMETRIC_KEY` | Chave simétrica de 32 bytes para PASETO |
| `ACCESS_TOKEN_DURATION` | Validade do access token (ex.: `15m`) |
| `REFRESH_TOKEN_DURATION` | Validade do refresh token (ex.: `168h`) |

> **Nota:** As variáveis também podem ser definidas como variáveis de ambiente do sistema — o Viper faz o bind automático.

---

## Rodando localmente

### 1. Subir o PostgreSQL

```bash
make postgres
make createdb
```

Isso cria um container `postgres16` na porta **5433** com usuário `root`, senha `secret` e banco `ground_guard`.

### 2. Rodar as migrations

```bash
make migrateup
```

### 3. Iniciar a API

```bash
make server
```

A API ficará disponível em `http://localhost:8080`.

---

## Rodando com Docker Compose

Sobe PostgreSQL e API juntos, com migrations automáticas na inicialização:

```bash
docker compose up --build
```

A API estará em `http://localhost:8081` (mapeada para a porta 8080 do container).

---

## Documentação da API (Swagger)

Com o servidor rodando, acesse:

```
http://localhost:8080/swagger/index.html
```

Para regenerar a documentação após alterar anotações nos handlers:

```bash
swag init -g cmd/server/main.go -o docs
```

---

## Endpoints principais

Base path: `/api/v1`

### Públicos (sem autenticação)

| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/users` | Criar conta |
| POST | `/users/login` | Login |
| POST | `/tokens/refresh` | Renovar access token |
| GET | `/faqs` | Listar FAQs |
| GET | `/faq/:uuid` | Obter FAQ |
| GET | `/help_contents` | Listar conteúdos de ajuda |
| GET | `/help_content/:uuid` | Obter conteúdo de ajuda |
| GET | `/tutorials` | Listar tutoriais |
| GET | `/tutorial/:uuid` | Obter tutorial |
| GET | `/legal-documents` | Listar documentos legais |
| GET | `/legal-document/:uuid` | Obter documento legal |

### Protegidos (requer `Authorization: Bearer {token}`)

| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/users/me` | Perfil do usuário autenticado |
| PUT | `/users/name/:uuid` | Atualizar nome |
| PUT | `/users/profile-image` | Atualizar foto de perfil |
| POST | `/devices` | Registrar dispositivo |
| GET | `/devices` | Listar dispositivos do usuário |
| GET | `/devices/:uuid` | Obter dispositivo |
| PUT | `/devices/link/:qr_token` | Vincular dispositivo via QR Code |
| PUT | `/devices/unlink/:uuid` | Desvincular dispositivo |
| PUT | `/devices/name/:uuid` | Renomear dispositivo |
| POST | `/irrigation_preference` | Criar preferência de irrigação |
| GET | `/irrigation_preference/:uuid` | Obter preferência |
| GET | `/irrigation_preference/device/:uuid` | Preferência por dispositivo |

Arquivos estáticos de perfil: `/uploads/profile/`
Arquivos estáticos de QR Code: `/uploads/qrcodes/`

---

## Comandos Make

| Comando | Descrição |
|---------|-----------|
| `make postgres` | Sobe container PostgreSQL 16 na porta 5433 |
| `make createdb` | Cria o banco `ground_guard` |
| `make dropdb` | Remove o banco `ground_guard` |
| `make migrateup` | Aplica todas as migrations |
| `make migrateup1` | Aplica uma migration |
| `make migratedown` | Reverte todas as migrations |
| `make migratedown1` | Reverte uma migration |
| `make sqlc` | Regenera código Go a partir das queries SQL |
| `make mock` | Regenera mocks do store (mockgen) |
| `make test` | Executa testes com cobertura |
| `make server` | Inicia o servidor em modo desenvolvimento |

---

## Testes

Certifique-se de que o PostgreSQL está rodando e as migrations foram aplicadas:

```bash
make test
```

O CI do GitHub Actions executa migrations e testes automaticamente em cada push/PR para a branch `main`.

---

## CI/CD

O workflow em `.github/workflows/ci.yml` executa:

1. Setup do Go 1.25
2. PostgreSQL como service container
3. `make migrateup`
4. `make test`

---

## Licença

MIT — veja detalhes no cabeçalho Swagger em `cmd/server/main.go`.

---

## Contato

**Guilherme Teixeira** — contato@groundguard.com
