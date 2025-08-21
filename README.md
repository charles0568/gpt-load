# GPT-Load

繁體中文文檔 | [English](README_EN.md)

[![Release](https://img.shields.io/github/v/release/tbphp/gpt-load)](https://github.com/tbphp/gpt-load/releases)
![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

一個高效能、企業級的 AI 介面透明代理服務，專門為需要整合多種 AI 服務的企業和開發者設計。採用 Go 語言開發，具備智慧密鑰管理、負載平衡和完善的監控功能，專為高併發生產環境而設計。

詳細請查看[官方文檔](https://www.gpt-load.com/docs)

<a href="https://hellogithub.com/repository/tbphp/gpt-load" target="_blank"><img src="https://api.hellogithub.com/v1/widgets/recommend.svg?rid=554dc4c46eb14092b9b0c56f1eb9021c&claim_uid=Qlh8vzrWJ0HCneG" alt="Featured｜HelloGitHub" style="width: 250px; height: 54px;" width="250" height="54" /></a>

## 功能特性

- **透明代理**: 完全保留原生 API 格式，支援 OpenAI、Google Gemini 和 Anthropic Claude 等多種格式
- **智慧密鑰管理**: 高效能密鑰池，支援分組管理、自動輪換和故障恢復
- **負載平衡**: 支援多上游端點的加權負載平衡，提升服務可用性
- **智慧故障處理**: 自動密鑰黑名單管理和恢復機制，確保服務連續性
- **動態配置**: 系統設定和分組配置支援熱重載，無需重啟即可生效
- **企業級架構**: 分散式主從部署，支援水平擴展和高可用
- **現代化管理**: 基於 Vue 3 的 Web 管理介面，直觀易用
- **全面監控**: 即時統計、健康檢查、詳細請求日誌
- **高效能設計**: 零拷貝串流傳輸、連線池複用、原子操作
- **生產就緒**: 優雅關閉、錯誤恢復、完善的安全機制
- **雙重認證體系**: 管理端與代理端認證分離，代理認證支援全域和分組級別密鑰

## 支援的 AI 服務

GPT-Load 作為透明代理服務，完整保留各 AI 服務商的原生 API 格式：

- **OpenAI 格式**: 官方 OpenAI API、Azure OpenAI、以及其他 OpenAI 相容服務
- **Google Gemini 格式**: Gemini Pro、Gemini Pro Vision 等模型的原生 API
- **Anthropic Claude 格式**: Claude 系列模型，支援高品質的對話和文字生成

## 快速開始

### 環境要求

- Go 1.23+ (原始碼建置)
- Docker (容器化部署)
- MySQL, PostgreSQL, 或 SQLite (資料庫儲存)
- Redis (快取和分散式協調，可選)

### 方式一：Docker 快速開始

```bash
docker run -d --name gpt-load \
    -p 3001:3001 \
    -e AUTH_KEY=sk-123456 \
    -v "$(pwd)/data":/app/data \
    ghcr.io/tbphp/gpt-load:latest
```

> 使用 `sk-123456` 登入管理介面：<http://localhost:3001>

### 方式二：使用 Docker Compose（推薦）

**安裝指令：**

```bash
# 建立目錄
mkdir -p gpt-load && cd gpt-load

# 下載配置檔案
wget https://raw.githubusercontent.com/tbphp/gpt-load/refs/heads/main/docker-compose.yml
wget -O .env https://raw.githubusercontent.com/tbphp/gpt-load/refs/heads/main/.env.example

# 啟動服務
docker compose up -d
```

預設安裝的是 SQLite 版本，適合輕量單機應用。

如需安裝 MySQL, PostgreSQL 及 Redis，請在 `docker-compose.yml` 檔案中取消所需服務的註解，並配置好對應的環境配置重啟即可。

**其他指令：**

```bash
# 查看服務狀態
docker compose ps

# 查看日誌
docker compose logs -f

# 重啟服務
docker compose down && docker compose up -d

# 更新到最新版本
docker compose pull && docker compose down && docker compose up -d
```

部署完成後：

- 存取 Web 管理介面：<http://localhost:3001>
- API 代理位址：<http://localhost:3001/proxy>

> 使用預設的認證 Key `sk-123456` 登入管理端，認證 Key 可以在 .env 中修改 AUTH_KEY。

### 方式三：原始碼建置

原始碼建置需要本地已安裝資料庫（SQLite、MySQL 或 PostgreSQL）和 Redis（可選）。

```bash
# 複製並建置
git clone https://github.com/tbphp/gpt-load.git
cd gpt-load
go mod tidy

# 建立配置
cp .env.example .env

# 修改 .env 中 DATABASE_DSN 和 REDIS_DSN 配置
# REDIS_DSN 為可選，如果不配置則啟用記憶體儲存

# 執行
make run
```

部署完成後：

- 存取 Web 管理介面：<http://localhost:3001>
- API 代理位址：<http://localhost:3001/proxy>

> 使用預設的認證 Key `sk-123456` 登入管理端，認證 Key 可以在 .env 中修改 AUTH_KEY。

### 方式四：叢集部署

叢集部署需要所有節點都連接同一個 MySQL（或者 PostgreSQL） 和 Redis，並且 Redis 是必須要求。建議使用統一的分散式 MySQL 和 Redis 叢集。

**部署要求：**

- 所有節點必須配置相同的 `AUTH_KEY`、`DATABASE_DSN`、`REDIS_DSN`
- 一主多從架構，從節點必須配置環境變數：`IS_SLAVE=true`

詳細請參考[叢集部署文檔](https://www.gpt-load.com/docs/cluster)

## 配置系統

### 配置架構概述

GPT-Load 採用雙層配置架構：

#### 1. 靜態配置（環境變數）

- **特點**：應用啟動時讀取，執行期間不可修改，需重啟應用生效
- **用途**：基礎設施配置，如資料庫連線、伺服器埠、認證密鑰等
- **管理方式**：透過 `.env` 檔案或系統環境變數設定

#### 2. 動態配置（熱重載）

- **系統設定**：儲存在資料庫中，為整個應用提供統一的行為基準
- **分組配置**：為特定分組定製的行為參數，可覆蓋系統設定
- **配置優先順序**：分組配置 > 系統設定 > 環境配置
- **特點**：支援熱重載，修改後立即生效，無需重啟應用

<details>
<summary>靜態配置（環境變數）</summary>

**伺服器配置：**

| 配置項       | 環境變數                           | 預設值          | 說明                       |
| ------------ | ---------------------------------- | --------------- | -------------------------- |
| 服務埠       | `PORT`                             | 3001            | HTTP 伺服器監聽埠          |
| 服務位址     | `HOST`                             | 0.0.0.0         | HTTP 伺服器綁定位址        |
| 讀取逾時     | `SERVER_READ_TIMEOUT`              | 60              | HTTP 伺服器讀取逾時（秒）  |
| 寫入逾時     | `SERVER_WRITE_TIMEOUT`             | 600             | HTTP 伺服器寫入逾時（秒）  |
| 閒置逾時     | `SERVER_IDLE_TIMEOUT`              | 120             | HTTP 連線閒置逾時（秒）    |
| 優雅關閉逾時 | `SERVER_GRACEFUL_SHUTDOWN_TIMEOUT` | 10              | 服務優雅關閉等待時間（秒） |
| 從節點模式   | `IS_SLAVE`                         | false           | 叢集部署時從節點標識       |
| 時區         | `TZ`                               | `Asia/Shanghai` | 指定時區                   |

**認證與資料庫配置：**

| 配置項       | 環境變數       | 預設值             | 說明                                   |
| ------------ | -------------- | ------------------ | -------------------------------------- |
| 管理密鑰     | `AUTH_KEY`     | `sk-123456`        | **管理端**的存取認證密鑰               |
| 資料庫連線   | `DATABASE_DSN` | ./data/gpt-load.db | 資料庫連線字串 (DSN) 或檔案路徑        |
| Redis 連線   | `REDIS_DSN`    | -                  | Redis 連線字串，為空時使用記憶體儲存   |

**效能與跨域配置：**

| 配置項       | 環境變數                  | 預設值                        | 說明                     |
| ------------ | ------------------------- | ----------------------------- | ------------------------ |
| 最大併發請求 | `MAX_CONCURRENT_REQUESTS` | 100                           | 系統允許的最大併發請求數 |
| 啟用 CORS    | `ENABLE_CORS`             | true                          | 是否啟用跨域資源共享     |
| 允許的來源   | `ALLOWED_ORIGINS`         | `*`                           | 允許的來源，逗號分隔     |
| 允許的方法   | `ALLOWED_METHODS`         | `GET,POST,PUT,DELETE,OPTIONS` | 允許的 HTTP 方法         |
| 允許的標頭   | `ALLOWED_HEADERS`         | `*`                           | 允許的請求標頭，逗號分隔 |
| 允許憑證     | `ALLOW_CREDENTIALS`       | false                         | 是否允許傳送憑證         |

**日誌配置：**

| 配置項       | 環境變數          | 預設值                | 說明                               |
| ------------ | ----------------- | --------------------- | ---------------------------------- |
| 日誌級別     | `LOG_LEVEL`       | `info`                | 日誌級別：debug, info, warn, error |
| 日誌格式     | `LOG_FORMAT`      | `text`                | 日誌格式：text, json               |
| 啟用檔案日誌 | `LOG_ENABLE_FILE` | false                 | 是否啟用檔案日誌輸出               |
| 日誌檔案路徑 | `LOG_FILE_PATH`   | `./data/logs/app.log` | 日誌檔案儲存路徑                   |

**代理配置：**

GPT-Load 會自動從環境變數中讀取代理設定，用於向上游 AI 服務商發起請求。

| 配置項     | 環境變數      | 預設值 | 說明                                     |
| ---------- | ------------- | ------ | ---------------------------------------- |
| HTTP 代理  | `HTTP_PROXY`  | -      | 用於 HTTP 請求的代理伺服器位址           |
| HTTPS 代理 | `HTTPS_PROXY` | -      | 用於 HTTPS 請求的代理伺服器位址          |
| 無代理     | `NO_PROXY`    | -      | 不需要透過代理存取的主機或網域名，逗號分隔 |

支援的代理協定格式：

- **HTTP**: `http://user:pass@host:port`
- **HTTPS**: `https://user:pass@host:port`
- **SOCKS5**: `socks5://user:pass@host:port`
</details>

<details>
<summary>動態配置（熱重載）</summary>

**基礎設定：**

| 配置項       | 欄位名                               | 預設值                      | 分組可覆蓋 | 說明                                   |
| ------------ | ------------------------------------ | --------------------------- | ---------- | -------------------------------------- |
| 專案位址     | `app_url`                            | `http://localhost:3001`     | ❌         | 專案基礎 URL                           |
| 日誌保留天數 | `request_log_retention_days`         | 7                           | ❌         | 請求日誌保留天數，0 為不清理           |
| 日誌寫入間隔 | `request_log_write_interval_minutes` | 1                           | ❌         | 日誌寫入資料庫週期（分鐘）             |
| 全域代理密鑰 | `proxy_keys`                         | 初始值為環境配置的 AUTH_KEY | ❌         | 全域生效的代理認證密鑰，多個用逗號分隔 |

**請求設定：**

| 配置項               | 欄位名                    | 預設值 | 分組可覆蓋 | 說明                           |
| -------------------- | ------------------------- | ------ | ---------- | ------------------------------ |
| 請求逾時             | `request_timeout`         | 600    | ✅         | 轉發請求完整生命週期逾時（秒） |
| 連線逾時             | `connect_timeout`         | 15     | ✅         | 與上游服務建立連線逾時（秒）   |
| 閒置連線逾時         | `idle_conn_timeout`       | 120    | ✅         | HTTP 用戶端閒置連線逾時（秒）  |
| 回應標頭逾時         | `response_header_timeout` | 600    | ✅         | 等待上游回應標頭逾時（秒）     |
| 最大閒置連線數       | `max_idle_conns`          | 100    | ✅         | 連線池最大閒置連線總數         |
| 每主機最大閒置連線數 | `max_idle_conns_per_host` | 50     | ✅         | 每個上游主機最大閒置連線數     |
| 代理伺服器位址       | `proxy_url`               | -      | ✅         | 用於轉發請求的 HTTP/HTTPS 代理，為空則使用環境配置 |

**密鑰配置：**

| 配置項         | 欄位名                            | 預設值 | 分組可覆蓋 | 說明                                             |
| -------------- | --------------------------------- | ------ | ---------- | ------------------------------------------------ |
| 最大重試次數   | `max_retries`                     | 3      | ✅         | 單個請求使用不同密鑰的最大重試次數               |
| 黑名單閾值     | `blacklist_threshold`             | 3      | ✅         | 密鑰連續失敗多少次後進入黑名單                   |
| 密鑰驗證間隔   | `key_validation_interval_minutes` | 60     | ✅         | 後台定時驗證密鑰週期（分鐘）                     |
| 密鑰驗證併發數 | `key_validation_concurrency`      | 10     | ✅         | 後台定時驗證無效 Key 時的併發數                  |
| 密鑰驗證逾時   | `key_validation_timeout_seconds`  | 20     | ✅         | 後台定時驗證單個 Key 時的 API 請求逾時時間（秒） |

</details>

## Web 管理介面

存取管理控制台：<http://localhost:3001>（預設位址）

### 介面展示

<img src="screenshot/dashboard.png" alt="儀表板" width="600" />

<br/>

<img src="screenshot/keys.png" alt="密鑰管理" width="600" />

<br/>

Web 管理介面提供以下功能：

- **儀表板**: 即時統計資訊和系統狀態概覽
- **密鑰管理**: 建立和配置 AI 服務商分組，新增、刪除和監控 API 密鑰
- **請求日誌**: 詳細的請求歷史記錄和除錯資訊
- **系統設定**: 全域配置管理和熱重載

## API 使用說明

<details>
<summary>代理介面呼叫方式</summary>

GPT-Load 透過分組名稱路由請求到不同的 AI 服務。使用方式如下：

#### 1. 代理端點格式

```text
http://localhost:3001/proxy/{group_name}/{原始API路徑}
```

- `{group_name}`: 在管理介面建立的分組名稱
- `{原始API路徑}`: 保持與原始 AI 服務完全一致的路徑

#### 2. 認證方式

在 Web 管理介面中配置**代理密鑰** (`Proxy Keys`)，可設定系統級別和分組級別的代理密鑰。

- **認證方式**: 與原生 API 一致，但需將原始密鑰替換為配置的代理密鑰。
- **密鑰作用域**: 在系統設定配置的 **全域代理密鑰** 可以在所有分組使用，在分組配置的 **分組代理密鑰** 僅在當前分組有效。
- **格式**: 多個密鑰使用半形英文逗號分隔。

#### 3. OpenAI 介面呼叫範例

假設建立了名為 `openai` 的分組：

**原始呼叫方式：**

```bash
curl -X POST https://api.openai.com/v1/chat/completions \
  -H "Authorization: Bearer sk-your-openai-key" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4.1-mini", "messages": [{"role": "user", "content": "Hello"}]}'
```

**代理呼叫方式：**

```bash
curl -X POST http://localhost:3001/proxy/openai/v1/chat/completions \
  -H "Authorization: Bearer your-proxy-key" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4.1-mini", "messages": [{"role": "user", "content": "Hello"}]}'
```

**变更说明：**

- 将 `https://api.openai.com` 替换为 `http://localhost:3001/proxy/openai`
- 将原始 API Key 替换为**代理密钥**

#### 4. Gemini 接口调用示例

假设创建了名为 `gemini` 的分组：

**原始调用方式：**

```bash
curl -X POST https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-pro:generateContent?key=your-gemini-key \
  -H "Content-Type: application/json" \
  -d '{"contents": [{"parts": [{"text": "Hello"}]}]}'
```

**代理调用方式：**

```bash
curl -X POST http://localhost:3001/proxy/gemini/v1beta/models/gemini-2.5-pro:generateContent?key=your-proxy-key \
  -H "Content-Type: application/json" \
  -d '{"contents": [{"parts": [{"text": "Hello"}]}]}'
```

**变更说明：**

- 将 `https://generativelanguage.googleapis.com` 替换为 `http://localhost:3001/proxy/gemini`
- 将 URL 参数中的 `key=your-gemini-key` 替换为**代理密钥**

#### 5. Anthropic 接口调用示例

假设创建了名为 `anthropic` 的分组：

**原始调用方式：**

```bash
curl -X POST https://api.anthropic.com/v1/messages \
  -H "x-api-key: sk-ant-api03-your-anthropic-key" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{"model": "claude-sonnet-4-20250514", "messages": [{"role": "user", "content": "Hello"}]}'
```

**代理调用方式：**

```bash
curl -X POST http://localhost:3001/proxy/anthropic/v1/messages \
  -H "x-api-key: your-proxy-key" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{"model": "claude-sonnet-4-20250514", "messages": [{"role": "user", "content": "Hello"}]}'
```

**变更说明：**

- 将 `https://api.anthropic.com` 替换为 `http://localhost:3001/proxy/anthropic`
- 将 `x-api-key` 头部中的原始 API Key 替换为**代理密钥**

#### 6. 支持的接口

**OpenAI 格式：**

- `/v1/chat/completions` - 聊天对话
- `/v1/completions` - 文本补全
- `/v1/embeddings` - 文本嵌入
- `/v1/models` - 模型列表
- 以及其他所有 OpenAI 兼容接口

**Gemini 格式：**

- `/v1beta/models/*/generateContent` - 内容生成
- `/v1beta/models` - 模型列表
- 以及其他所有 Gemini 原生接口

**Anthropic 格式：**

- `/v1/messages` - 消息对话
- `/v1/models` - 模型列表（如果可用）
- 以及其他所有 Anthropic 原生接口

#### 7. 客户端 SDK 配置

**OpenAI Python SDK：**

```python
from openai import OpenAI

client = OpenAI(
    api_key="your-proxy-key",  # 使用密钥
    base_url="http://localhost:3001/proxy/openai"  # 使用代理端点
)

response = client.chat.completions.create(
    model="gpt-4.1-mini",
    messages=[{"role": "user", "content": "Hello"}]
)
```

**Google Gemini SDK (Python)：**

```python
import google.generativeai as genai

# 配置 API 密钥和基础 URL
genai.configure(
    api_key="your-proxy-key",  # 使用代理密钥
    client_options={"api_endpoint": "http://localhost:3001/proxy/gemini"}
)

model = genai.GenerativeModel('gemini-2.5-pro')
response = model.generate_content("Hello")
```

**Anthropic SDK (Python)：**

```python
from anthropic import Anthropic

client = Anthropic(
    api_key="your-proxy-key",  # 使用代理密钥
    base_url="http://localhost:3001/proxy/anthropic"  # 使用代理端点
)

response = client.messages.create(
    model="claude-sonnet-4-20250514",
    messages=[{"role": "user", "content": "Hello"}]
)
```

> **重要提示**：作為透明代理服務，GPT-Load 完全保留各 AI 服務的原生 API 格式和認證方式，僅需要替換端點位址並使用在管理端配置的**代理密鑰**即可無縫遷移。

</details>

## 授權條款

MIT 授權條款 - 詳情請參閱 [LICENSE](LICENSE) 檔案。

## Star History

[![Stargazers over time](https://starchart.cc/tbphp/gpt-load.svg?variant=adaptive)](https://starchart.cc/tbphp/gpt-load)
