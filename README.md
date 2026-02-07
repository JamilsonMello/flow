# Flow Framework

Distribuição e conformidade de contratos para testes E2E.

## Requisitos

- Docker & Docker Compose
- Go 1.18+

## Instalação e Execução

### 1. Iniciar Infraestrutura

Suba o banco de dados PostgreSQL e o PGAdmin:

```bash
docker-compose up -d
```

O banco criado será `flow_db` e a tabela `flow_events` será inicializada automaticamente.

### 2. Rodar Aplicações de Exemplo

O projeto contém dois serviços de exemplo simulando um fluxo de Pedido.

**Passo 1: Service A (Inicia o fluxo e cria a promessa)**

```bash
go run cmd/service-a/main.go
```
*Saída esperada: "Service A completed. Request 'sent' to Service B."*

**Passo 2: Service B (Processa, asserta e finaliza)**

```bash
go run cmd/service-b/main.go
```
*Saída esperada: "✅ Flow validation PASSED!"*

### 3. Verificar Dados no PGAdmin

- **URL**: http://localhost:8080
- **Email**: admin@admin.com
- **Senha**: admin
- **Server Host**: postgres (ou host.docker.internal)
- **Username**: user
- **Password**: password

## Estrutura do Projeto

- `pkg/flow`: Core SDK (Start, CreatePoint, Finish, etc).
- `cmd/service-a`: Simulação do sistema de origem.
- `cmd/service-b`: Simulação do sistema de destino e worker de validação.
