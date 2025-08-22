# Google Gemini API 配額限制解決方案

## 當前問題
- 錯誤：Quota exceeded for quota metric 'Generate Content API requests per minute'
- 專案：535213709716
- 服務：generativelanguage.googleapis.com

## Google Gemini API 免費配額限制

### 免費版限制
- **每分鐘請求數 (RPM)**：15 requests/minute
- **每天請求數 (RPD)**：1,500 requests/day  
- **每分鐘 Token 數 (TPM)**：32,000 tokens/minute
- **每天 Token 數 (TPD)**：50,000 tokens/day

### 付費版限制
- **每分鐘請求數 (RPM)**：1,000 requests/minute
- **每天請求數 (RPD)**：50,000 requests/day
- **每分鐘 Token 數 (TPM)**：4,000,000 tokens/minute

## GPT-Load 優化配置

### 系統設定建議
```
基礎參數：
- 請求日誌保留天數：3 天
- 日誌延遲寫入週期：5 分鐘

請求設定：
- 請求逾時：180 秒
- 連線逾時：10 秒
- 最大重試次數：1 次

密鑰配置：
- 最大重試次數：1 次
- 黑名單閾值：15 次
- 密鑰驗證間隔：360 分鐘
- 密鑰驗證併發數：1
- 密鑰驗證逾時：15 秒
```

### 分組配置建議
```
分組名稱：gemini-free
上游地址：https://generativelanguage.googleapis.com
請求逾時：120 秒
最大重試次數：0 次
黑名單閾值：20 次
```

## 多專案策略

### 建立多個 Google Cloud 專案
1. 前往 https://console.cloud.google.com/
2. 建立新專案（建議 3-5 個）
3. 為每個專案啟用 Generative Language API
4. 生成各自的 API 密鑰

### 專案配額分散
- 專案 A：處理 1-500 請求/天
- 專案 B：處理 501-1000 請求/天  
- 專案 C：處理 1001-1500 請求/天

## 替代方案

### OpenAI API 配置
```
分組名稱：openai
上游地址：https://api.openai.com
模型：gpt-4o-mini (便宜), gpt-4o (高品質)
優點：更寬鬆的配額限制
```

### Anthropic Claude 配置
```
分組名稱：claude
上游地址：https://api.anthropic.com  
模型：claude-3-haiku (快速), claude-3-sonnet (平衡)
優點：高品質回應，合理配額
```

### 本地模型配置
```
分組名稱：ollama
上游地址：http://localhost:11434
模型：llama3.1, qwen2.5
優點：無配額限制，完全私有
```

## 監控和警報

### 在 GPT-Load 中監控
1. 儀表板 - 查看請求成功率
2. 日誌頁面 - 監控 429 錯誤頻率
3. 密鑰管理 - 檢查密鑰狀態

### 配額使用監控
1. Google Cloud Console - API 配額頁面
2. 設定配額警報（80% 使用量時通知）
3. 定期檢查配額重置時間

## 緊急處理步驟

### 當遇到 429 錯誤時
1. **立即停止測試**：避免進一步消耗配額
2. **等待配額重置**：通常每分鐘重置
3. **切換到備用服務**：使用 OpenAI 或 Claude
4. **檢查配額狀態**：在 Google Cloud Console 中查看

### 長期解決方案
1. **升級到付費方案**：獲得更高配額
2. **實施多服務策略**：不依賴單一 AI 服務
3. **優化請求頻率**：避免短時間大量請求
4. **使用快取機制**：減少重複請求
