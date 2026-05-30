# p2p-ranking-board

被動追蹤海盜灣（The Pirate Bay）榜單:定期抓取指定類別的 top-100、比對上次快照、把變動印出來。

設計上是**無狀態的一次性命令列工具** —— 排程不內建,交給外部代理（例如 Hermes 的 cron）定時觸發。「哪些變動值得通知我」這種語意判斷留給外部代理的 LLM 層;本程式只做確定性的抓取、比對、輸出。

合法用途範例:追蹤公眾領域影片、開源軟體、Linux 發行版的散布榜單。是否散布特定內容、是否取得授權,由使用者自行負責。

## 建置

> 需要 Go 1.25 以上。

```sh
go build -o p2p-ranking-board ./cmd/p2p-ranking-board
```

## 使用

```sh
# 抓取 + 比對 + 印出變動(首次執行只建立基準,不報變動)
p2p-ranking-board poll

# 給外部代理讀的結構化輸出,並只關心新進榜
p2p-ranking-board poll --json --only new

# 把核准的條目(用 poll 輸出裡的 id)解析成磁力連結,印到 stdout
p2p-ranking-board get 207:4a3f5e08bcef
```

### `poll` 旗標

| 旗標 | 預設 | 說明 |
|------|------|------|
| `--state` | XDG 狀態目錄 | 快照檔路徑 |
| `--json` | false | 變動以 JSON 輸出（供 LLM／代理解析） |
| `--only` | 全部 | 只報這些類型:`new,dropped,rank_move,seed_shift` |
| `--seed-threshold` | 0.20 | 種子數變動達此比例才報 `seed_shift` |
| `--timeout` | 60s | 抓取整體預算 |

### 追蹤類別

第一版固定追蹤 apibay 類別 `207`（HD 電影）、`208`（HD 影集）、`300`/`301`（應用程式／Windows）、`303`（Linux／UNIX）、`401`（PC 遊戲）、`601`（電子書）,各取 top-100。

## 用 Hermes cron 觸發

讓 Hermes 的 cron 定時跑 `poll --json`,把標準輸出的變動交給後續 LLM 步驟判讀,再決定要不要通知你:

```cron
# 每 30 分鐘輪詢一次
*/30 * * * * cd /path/to/p2p-ranking-board && ./p2p-ranking-board poll --json --only new,rank_move
```

本程式只負責「抓→比→印」;判斷與通知由 Hermes 那層完成,符合「Go 純確定性、判斷留代理」的分工。

## 快照

狀態存成單一 JSON 檔（預設 `${XDG_STATE_HOME:-~/.local/state}/p2p-ranking-board/snapshot.json`),以「類別:資訊雜湊」為鍵。寫入採先寫暫存檔再更名的原子操作。任一類別抓取失敗時**不覆寫**既有快照、整體回非零,等下一輪重試。刪除快照即重建基準。

## 設計邊界:只提供磁力連結,不下載

本工具**只**做資料來源:`get <id>` 把磁力連結印到標準輸出就結束,不驗證、不下載、不呼叫任何程式,也不知道下游是誰。要驗證健康度或下載,自行用 shell 管道把標準輸出接給別的程式即可,本工具完全不碰下載設定。

## 列出當前榜單

```sh
p2p-ranking-board list -n 10   # 各追蹤類別前 10 名(排名、種子數、名稱、識別碼),不碰快照
```

挑到要的就用輸出的識別碼接 `get <id>` 取得磁力連結。
