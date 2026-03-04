# TDD 開發執行指南 (AI Agent 專用)

本指南定義 AI Agent 在執行 TDD（測試驅動開發）時應遵循的流程、互動模式與回報機制，並已針對 Go 語言專案進行最佳化。

---

## 1. TDD 核心循環

AI Agent 必須遵循 **Red-Green-Refactor** 循環：

```
┌─────────────────────────────────────────────────────────┐
│  1. RED    │ 寫一個失敗的測試 (預期失敗)                  │
│────────────┼────────────────────────────────────────────│
│  2. GREEN  │ 寫最少的生產程式碼讓測試通過                 │
│────────────┼────────────────────────────────────────────│
│  3. REFACTOR │ 重構程式碼 (保持測試通過)                 │
└─────────────────────────────────────────────────────────┘
         ↑___________________________________↓
              重複循環直到功能完成
```

---

## 2. Task 拆分原則

每個任務應拆分為 **可測試的子任務**：

| 拆分標準 | 說明                                      |
| -------- | ----------------------------------------- |
| 單一職責 | 每個 function/struct 只做一件事           |
| 可測試性 | 輸出可斷言、副作用可隔離 (善用 Interface) |
| 邊界條件 | 包含正常、邊界、錯誤情況                  |

### Task 範例模板

```markdown
## Task: [功能名稱]

### 子任務

- [ ] T1: 寫測試 - [測試項目描述]
- [ ] T2: 實作 - [功能描述]
- [ ] T3: 重構 - [優化點]
- [ ] T4: 整合測試 - [端到端場景]
```

---

## 3. 互動回報機制

### 3.1 回報時機

每個 **子任務 (Sub-task)** 開始與完成時必須回報：

```
📋 [開始] T1: 寫測試 - 驗證使用者登入功能
   └─ 預期：建立 TestLogin_Success, TestLogin_Failure, TestLogin_InvalidPassword

✅ [完成] T1: 寫測試 - 通過 3/3 測試
   └─ 紅燈：符合預期
```

### 3.2 回報模板

```markdown
## 🚀 Task 進度回報

### 任務：[功能名稱]

**狀態**：進行中 / 完成

### 子任務進度

| 子任務       | 狀態      | 說明           |
| ------------ | --------- | -------------- |
| T1: 寫測試   | ✅ 完成   | 3 tests passed |
| T2: 實作     | 🔄 進行中 | 實作中...      |
| T3: 重構     | ⏳ 待處理 | -              |
| T4: 整合測試 | ⏳ 待處理 | -              |

### 變更檔案

- `pkg/auth/auth.go` - 新增
- `pkg/auth/auth_test.go` - 新增
- `pkg/utils/utils.go` - 修改

### 遇到問題

無 / [問題描述 + 詢問選項]
```

### 3.3 詢問時機

以下情況 **必須** 詢問開發者：

| 情況       | 詢問範例                                                     |
| ---------- | ------------------------------------------------------------ |
| 需求不明確 | 「請問登入失敗時要回傳特定錯誤碼還是通用 error？」           |
| 技術決策   | 「要用 gomock 還是 testify/mock？這個單元測試適合哪種？」    |
| 測試策略   | 「這個整合測試是否需要 mock 外部 API，還是使用 httptest？」  |
| 優先順序   | 「有兩個功能都要做要先做哪個？」                             |
| 發現風險   | 「重構可能影響既有介面 (Interface)，是否要更新所有調用處？」 |

---

## 4. 測試寫作規範 (Go 專屬)

### 4.1 表格驅動測試 (Table-Driven Tests) 文件結構

在 Go 語言中，強烈建議使用表格驅動測試來覆蓋多種場景：

```go
// pkg/auth/auth_test.go
package auth_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "minibot/pkg/auth"
)

func TestAuthService_Login(t *testing.T) {
    // 建立測試主體 (Arrange)
    service := auth.NewAuthService()

    tests := []struct {
        name          string
        username      string
        password      string
        expectedError string
        expectedToken bool
    }{
        {
            name:          "正常情況：正確帳密應回傳 Token",
            username:      "testuser",
            password:      "validpass",
            expectedError: "",
            expectedToken: true,
        },
        {
            name:          "邊界情況：空帳號應回傳錯誤",
            username:      "",
            password:      "validpass",
            expectedError: "username is required",
            expectedToken: false,
        },
        {
            name:          "錯誤情況：密碼錯誤",
            username:      "testuser",
            password:      "wrongpass",
            expectedError: "invalid credentials",
            expectedToken: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Act
            token, err := service.Login(tt.username, tt.password)

            // Assert
            if tt.expectedError != "" {
                assert.EqualError(t, err, tt.expectedError)
                assert.Empty(t, token)
            } else {
                assert.NoError(t, err)
                assert.NotEmpty(t, token)
            }
        })
    }
}
```

### 4.2 命名規範

| 類型           | 命名模式                        | 範例                               |
| -------------- | ------------------------------- | ---------------------------------- |
| 測試檔案       | `<module>_test.go`              | `auth_test.go`                     |
| 測試套件/主體  | `Test<StructName>`              | `TestAuthService`                  |
| 測試方法       | `Test<StructName>_<MethodName>` | `TestAuthService_Login`            |
| 子測試 (Table) | `<場景說明>`                    | `"正常情況：正確帳密應回傳 Token"` |

### 4.3 測試隔離原則

- **每個測試獨立**：不依賴其他測試的執行順序 (可用 `t.Parallel()` 驗證)。
- **無狀態共享**：在每個 `t.Run` 內重新初始化 mock 和變數，避免 goroutine 之間的狀態污染。

---

## 5. 驗證清單

### 5.1 Task 開始前

- [ ] 確認需求與驗收標準
- [ ] 拆分可測試的子任務
- [ ] 確認測試檔案位置 (`*_test.go`)

### 5.2 Red 階段 (寫測試)

- [ ] 寫失敗的測試 (預期 FAILED)
- [ ] 執行 `go test` 確認失敗原因是「邏輯未正確實作」而非編譯錯誤

### 5.3 Green 階段 (實作)

- [ ] 寫最少量程式碼通過測試
- [ ] 執行 `go test ./...` 確認相關套件全數通過

### 5.4 Refactor 階段

- [ ] 重構程式碼 (提取共用邏輯或介面)
- [ ] 確保測試仍然通過
- [ ] 檢查是否有重複程式碼

### 5.5 Task 完成前

- [ ] 執行完整測試套件 (`go test ./...`)
- [ ] 執行 Linter (`golangci-lint run`)
- [ ] 查看覆蓋率狀態 (`go test -coverprofile=coverage.out ./...`)
- [ ] 回報進度給開發者

---

## 6. 指令速查 (Go 工具鏈)

| 動作                       | 指令                                            |
| -------------------------- | ----------------------------------------------- |
| 執行所有測試               | `go test ./...`                                 |
| 執行特定套件               | `go test ./pkg/auth/...`                        |
| 執行特定測試               | `go test -run TestAuthService_Login ./pkg/auth` |
| 執行並顯示輸出             | `go test -v ./...`                              |
| 執行並顯示覆蓋率           | `go test -coverprofile=coverage.out ./...`      |
| 產生 HTML 覆蓋率報告       | `go tool cover -html=coverage.out`              |
| 執行整合測試 (如果有 Tags) | `go test -tags=integration ./...`               |

---

## 7. 推薦工具與架構

### 7.1 斷言庫 (Testify)

使用 `testify` 可以讓斷言更易讀，避免寫滿滿的 `if err != nil`：

```bash
go get github.com/stretchr/testify
```

```go
import "github.com/stretchr/testify/assert"

assert.Equal(t, "expected", actual)
assert.NoError(t, err)
```

### 7.2 依賴注入與 Mocking

在 Go 中，應透過 **Interface (介面)** 進行依賴注入，以利 Mocking：

```go
// 生產代碼定義 Interface
type UserRepository interface {
    GetUser(id string) (*User, error)
}
```

使用 `testify/mock` 來實作 Mock：

```go
import "github.com/stretchr/testify/mock"

type MockUserRepository struct {
    mock.Mock
}

func (m *MockUserRepository) GetUser(id string) (*User, error) {
    args := m.Called(id)
    if args.Get(0) != nil {
        return args.Get(0).(*User), args.Error(1)
    }
    return nil, args.Error(1)
}

// 在測試中使用
func TestService(t *testing.T) {
    mockRepo := new(MockUserRepository)
    mockRepo.On("GetUser", "123").Return(&User{Name: "test"}, nil)
    // 注入 mockRepo 到測試對象
}
```

### 7.3 Go Fuzz測試 (模糊測試)

Go 1.18 內建 Fuzz 支援，適合用來測試 edge cases：

```go
func FuzzParseInt(f *testing.F) {
    f.Add("123") // seed corpus
    f.Fuzz(func(t *testing.T, s string) {
        // 確保不會 panic 等基本性質
        ParseInt(s)
    })
}
```

執行指令: `go test -fuzz=Fuzz`

---

## 8. 範例：完整 Task 流程

### Task: 新增使用者登入功能

````markdown
## 🚀 Task 進度回報

### 任務：使用者登入功能

**狀態**：進行中

---

### T1: 寫測試 (RED)

**狀態**：✅ 完成

```text
$ go test -v ./pkg/auth
=== RUN   TestAuthService_Login
=== RUN   TestAuthService_Login/正常情況：正確帳密應回傳_Token
    auth_test.go:42:
        	Error Trace:	auth_test.go:42
        	Error:      	Not equal:
        	            	expected: "test-token"
        	            	actual  : ""
        	Test:       	TestAuthService_Login/正常情況：正確帳密應回傳_Token
--- FAIL: TestAuthService_Login (0.00s)
FAIL
FAIL	minibot/pkg/auth	0.013s
```
````

紅燈：符合預期 (功能未實作)

---

### T2: 實作登入功能 (GREEN)

**狀態**：🔄 進行中

正在實作 `AuthService.Login()` 方法...

````

```markdown
## 🚀 Task 進度回報

### 任務：使用者登入功能
**狀態**：✅ 完成

---

### T1: 寫測試 - ✅ 完成
- `auth_test.go` (Table-Driven Tests)
  - `正常情況：正確帳密應回傳 Token`
  - `邊界情況：空帳號應回傳錯誤`
  - `錯誤情況：密碼錯誤`

### T2: 實作 - ✅ 完成
- `AuthService.Login()` 已實作 (auth.go)
- `go test ./pkg/auth` - ok

### T3: 重構 - ✅ 完成
- 提取密碼 hash 驗證為獨立 unexported function

### T4: 整合測試 - ✅ 完成
- 使用 `httptest` 驗證登入 API 端點 - 通過

---

### 變更檔案
- `pkg/auth/auth.go` - 新增
- `pkg/auth/auth_test.go` - 新增

### 驗證結果
```text
$ go test -cover ./pkg/auth
ok  	minibot/pkg/auth	0.008s	coverage: 100.0% of statements
````

````

---

## 9. 開發者互動範例

### 範例 1: 需求確認

```text
❓ 詢問：關於登入失敗的處理方式，請確認：
1. 錯誤發生時是否回傳自定義的結構體 (如 `*AppError`)，還是直接 `errors.New`？
2. 失敗訊息要顯示「帳號或密碼錯誤」還是分開顯示？
````

### 範例 2: 技術決策

```text
❓ 詢問：在實作外部 API 請求時：
這個單元測試要自己實作 `http.RoundTripper` 還是用現成的 mock sever (例如 `httptest.NewServer`)？
專案有偏好哪一種方式嗎？
```

### 範例 3: 優先順序

```text
⚠️ 發現：重構 `AuthService` 可能影響現有的介面 `Authenticator` 的定義
需要連帶修改三個實作此介面的地方。
是否要：
1. 先重構並更新所有影響的地方（風險較高但架構更好）
2. 暫時保留舊介面，開設新介面？
```
