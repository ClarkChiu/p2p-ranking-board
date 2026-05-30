## 1. 專案骨架

- [x] 1.1 `go mod init github.com/freebooters/p2p-ranking-board`(Go 1.25+),建立 `cmd/` 與 `internal/{source,snapshot,diff,notify,resolve}` 目錄
- [x] 1.2 定義共用型別 `Entry{ InfoHash, Title, Category, Rank, Seeders, Leechers, SizeBytes, Added }` 與識別碼格式 `<cat>:<infohash前12碼>`

## 2. apibay 來源(對應 ranking-source）

- [x] 2.1 實作抓取 `precompiled/data_top100_<cat>.json`,用容忍式解碼(`json.Number`)吸收 int/string 欄位差異
- [x] 2.2 七個類別 `207,208,300,301,303,401,601` 並行抓取,逐類別逾時/錯誤獨立處理
- [x] 2.3 正規化為 `Entry`(資訊雜湊轉小寫十六進位、略過無效條目);任一類別失敗回明確錯誤且不得讓上層覆寫快照

## 3. 快照與差異(對應 change-detection）

- [x] 3.1 `Snapshot{ TakenAt, Entries map["<cat>:<infohash>"]Entry }` 的讀取/寫入,寫入採先寫 `*.tmp` 再 `rename` 的原子操作
- [x] 3.2 預設路徑 `${XDG_STATE_HOME:-~/.local/state}/p2p-ranking-board/snapshot.json`,可用 `--state` 覆寫
- [x] 3.3 純函式差異引擎:新進榜、離榜、排名變動、種子數變動(預設 ±20% 門檻);首次無快照時回空差異並寫入基準

## 4. 通知(對應 update-notification）

- [x] 4.1 定義 `Notifier` 介面與註冊機制
- [x] 4.2 實作 `stdoutNotifier`:人類可讀文字 + `--json` 結構化輸出(供 Hermes LLM 判讀)
- [x] 4.3 門檻過濾(`--only=new,rankjump,...`)在送進 Notifier 前套用;空差異不通知

## 5. 解析磁力連結(對應 magnet-resolve）

- [x] 5.1 `get <id>`:由最近快照解析識別碼並組磁力連結,印到標準輸出;識別碼不存在 / 無快照回明確錯誤
- [x] 5.2 標準輸出只含磁力連結(診斷走標準錯誤),可被 shell 管道接續;不執行下載、不相依任何下載程式
- [x] 5.3 `internal/resolve` 只負責 id→magnet;移除舊的 exec 交棒(`--download`/`--p2pscout-cmd`)

## 6. CLI 組裝與驗證

- [x] 6.1 `cmd/p2p-ranking-board`:組裝 `poll`(抓+比對+通知+寫快照)與 `get` 子命令與旗標
- [x] 6.2 `go build ./... && go vet ./...` 通過;對 apibay 跑一次真實 `poll` 煙霧測試(首次建基準、第二次比出差異)
- [x] 6.3 README:Hermes cron 觸發 `poll` 的接線範例與快照/旗標說明

## 7. 列出當前榜單(對應 ranking-list）

- [x] 7.1 `list` 子命令:逐類別抓 top100、印前 N 名(排名、種子數、名稱、識別碼),不讀寫快照
- [x] 7.2 `--top/-n` 旗標控制每類別筆數;任一類別失敗回明確錯誤
- [x] 7.3 移除冗餘類別 `300`(與 `301` 重複),`source.Categories` 改為 `207,208,301,303,401,601`
