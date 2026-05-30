# update-notification Specification

## Purpose
TBD - created by archiving change add-tpb-ranking-tracker. Update Purpose after archive.
## Requirements
### Requirement: 推播變動通知

系統 SHALL 把變動集合格式化為人類可讀的通知,透過設定的通知管道送出。每則通知 MUST 含條目名稱、變動類型、種子節點數,以及供後續核准用的條目識別碼。

#### Scenario: 有變動時送出通知
- **WHEN** 比對結果含一個以上的變動條目
- **THEN** 系統透過設定的管道送出通知,內容列出每個變動條目及其識別碼

#### Scenario: 無變動時不打擾
- **WHEN** 比對結果為空
- **THEN** 系統 SHALL NOT 送出任何通知

### Requirement: 可插拔通知管道

系統 SHALL 以介面抽象通知管道,讓管道可替換而不影響比對邏輯。第一版 MUST 至少提供一種可運作的管道。

#### Scenario: 切換通知管道
- **WHEN** 設定指定某個已註冊的通知管道
- **THEN** 系統 SHALL 透過該管道送出通知,無需修改比對或抓取程式碼

### Requirement: 通知門檻過濾

系統 SHALL 支援設定通知門檻,僅推播符合條件的變動(例如僅新進榜、或排名躍升達設定名次以上),以抑制雜訊。

#### Scenario: 過濾未達門檻的變動
- **WHEN** 設定門檻為「僅新進榜」且本次變動僅含種子數變動
- **THEN** 系統 SHALL NOT 送出通知

