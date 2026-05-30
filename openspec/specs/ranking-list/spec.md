# ranking-list Specification

## Purpose
TBD - created by archiving change add-tpb-ranking-tracker. Update Purpose after archive.
## Requirements
### Requirement: 列出當前榜單

系統 SHALL 提供一個不經快照比對、直接列出當前各追蹤類別前 N 名榜單條目的命令。每筆 MUST 顯示排名、種子節點數、名稱與識別碼,供使用者隨時查看現況並挑選要下載的條目。

#### Scenario: 列出每類別前 N 名
- **WHEN** 使用者要求列出且指定每類別筆數 N
- **THEN** 系統依序對每個追蹤類別輸出其前 N 名(不足 N 筆則輸出全部),每筆含排名、種子節點數、名稱與識別碼

#### Scenario: 不修改快照
- **WHEN** 使用者執行列出命令
- **THEN** 系統 SHALL NOT 讀取或寫入快照,也不產生任何「變動」或通知

#### Scenario: 資料源失敗
- **WHEN** 任一類別抓取失敗
- **THEN** 系統 SHALL 回傳明確錯誤

