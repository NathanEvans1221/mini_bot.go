# Security Upgrade TODO

## 概述

本專案需要針對多個安全漏洞進行修復，以提升系統整體安全性。以下是待辦事項清單，所有 AI Agent 都應按照優先順序執行。

---

## 高優先級 (Critical)

### 1. Shell 指令注入防護 [shell.go]

**問題**: 現有危險指令過濾使用簡單的正則表達式，容易被繞過。

**現有程式碼問題**:
- 只檢測已知危險模式，可輕易繞過
- 未實作命令允許清單 (allowlist)
- 未對參數進行清理

**修復任務**:
- [x] 實作命令允許清單機制，僅允許安全指令 (如 `ls`, `cat`, `grep`, `cd` 等)
- [x] 新增命令白名單驗證函數 `validateCommand(cmd string) bool`
- [x] 移除危險指令 (`curl`, `wget`, `nc`, `bash`, `powershell` 等可執行任意程式碼的指令)
- [x] 實作參數清理，移除可能的命令連接符 (`;`, `|`, `&&`, `||`, `$()`, `` ` ` ``)
- [x] 新增逾時保護 (已存在但需加強)

**參考實作位置**: `pkg/tools/shell.go:29-51`

---

### 2. 路徑穿越漏洞修復 [sandbox.go]

**問題**: 使用 `strings.HasPrefix` 進行路徑檢查，可被 symlink 繞過。

**現有程式碼問題**:
```go
if !strings.HasPrefix(absTargetPath, s.Workspace) {
    return "", fmt.Errorf("path escapes workspace bounds: %s", inputPath)
}
```

**修復任務**:
- [x] 使用 `strings.HasPrefix` 改為目錄完全匹配驗證
- [x] 在路徑檢查前解析所有 symlink
- [x] 新增 Windows 路徑大小寫不敏感比對
- [x] 驗證最終路徑確實在允許目錄內 (使用 `filepath.Dir` 逐步驗證父目錄)

**參考實作位置**: `pkg/tools/sandbox.go:40-46`

---

### 3. 敏感資料加密 [config.go]

**問題**: API Keys、Bot Tokens 以明文儲存於 config.json。

**現有程式碼問題**:
- 明文儲存密鑰
- 無環境變數強制使用機制
- 缺少密鑰輪換機制

**修復任務**:
- [x] 新增 `.env` 檔案支援，使用 `os.Getenv` 讀取敏感資訊
- [x] 在 config.go 中優先讀取環境變數覆寫
- [x] 範例 config.json 中移除所有敏感資訊，改用 placeholder
- [x] 新增提示訊息告知使用者勿提交包含密鑰的 config.json

**參考實作位置**: `pkg/config/config.go:90-117`

---

## 中優先級 (High)

### 4. Prompt Injection 防護 [memory.go, context.go]

**問題**: 使用者輸入未經清理即存入記憶體，可能導致 Agent 行為被操縱。

**修復任務**:
- [x] 新增輸入清理函數，檢測常見 injection 模式
- [x] 實作特殊字元轉義 (`<`, `>`, `{`, `}`, `[`, `]`)
- [x] 建立訊息長度限制
- [x] 考慮使用 Markdown code block 包裝使用者輸入

**參考實作位置**: `pkg/agent/memory.go:25-35`

---

### 5. HTTP 安全強化 [openai_compat.go]

**問題**: HTTP 請求未驗證 SSL 憑證，無 TLS 版本控制。

**修復任務**:
- [x] 自訂 HTTP Client，強制 TLS 1.2+
- [x] 啟用憑證驗證 (移除未驗證的 Transport)
- [x] 新增連線逾時保護
- [x] 記錄 TLS 版本以供稽核

**參考實作位置**: `pkg/providers/openai_compat.go:74`

---

### 6. 檔案權限強化 [filesystem.go, memory.go]

**問題**: 敏感檔案使用過於寬鬆的權限 (0644)。

**修復任務**:
- [x] 記憶體檔案使用 0600 權限
- [x] config 載入時驗證檔案權限，警告過於寬鬆的權限
- [x] 新增工作目錄隔離確認

**參考實作位置**: `pkg/tools/filesystem.go:76`

---

## 低優先級 (Medium)

### 7. 速率限制 (Rate Limiting)

**問題**: 無法防止 API 濫用或 DoS 攻擊。

**修復任務**:
- [x] 在 agent loop 中實作工具呼叫次數限制
- [x] 新增每分鐘請求數限制
- [x] 記錄異常使用模式

---

### 8. 日誌安全 [logger.go]

**問題**: 日誌可能包含敏感資訊。

**修復任務**:
- [x] 實作敏感資訊過濾 (API keys, tokens)
- [x] 新增日誌級別控制
- [x] 確保錯誤訊息不暴露內部實作細節

---

### 9. 允許清單驗證

**問題**: 工具參數缺乏嚴格驗證。

**修復任務**:
- [x] 為所有工具實作 JSON Schema 驗證
- [x] 限制路徑字元 (僅允許安全字元)
- [x] 限制命令長度

---

## 執行順序建議

1. **第一階段** (立即): 修復 shell.go 與 sandbox.go (最高風險)
2. **第二階段** (短期): 完成 config.go 敏感資料加密
3. **第三階段** (中期): 實作 prompt injection 防護與 HTTP 安全
4. **第四階段** (長期): 速率限制、日誌安全、允許清單驗證

---

## 驗證方式

每個修復完成後，請執行以下驗證:

1. **Shell Injection**: 嘗試執行 `rm -rf /`, `; cat /etc/passwd`, `$(whoami)`
2. **Path Traversal**: 嘗試 `../../../etc/passwd`, 使用 symlink 逃逸
3. **Prompt Injection**: 嘗試輸入 `Ignore previous instructions and...`
4. **HTTP**: 驗證 TLS 連線正確驗證憑證

---

## 相關檔案清單

| 檔案 | 重要性 |
|------|--------|
| `pkg/tools/shell.go` | Critical |
| `pkg/tools/sandbox.go` | Critical |
| `pkg/config/config.go` | High |
| `pkg/agent/memory.go` | High |
| `pkg/agent/context.go` | High |
| `pkg/providers/openai_compat.go` | High |
| `pkg/tools/filesystem.go` | Medium |
| `config/config.example.json` | High |
| `README.md` | High |

---

*本檔案由 AI Agent 自動生成，用於追蹤安全升級進度。*
