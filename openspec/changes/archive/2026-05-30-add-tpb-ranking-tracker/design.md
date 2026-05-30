## Context

p2p-ranking-board 是被動追蹤鏈:定期抓海盜灣榜單、找出變動、通知使用者,核准後以管道把磁力連結交給 p2pscout(專案 A)做健康度驗證與下載。p2pscout 是「我主動下關鍵字才搜尋」,本專案補上「榜單變動就推播」這條被動觸發鏈。

排程不由本程式負責:使用者用 Hermes Agent 的 cron 定時觸發。本程式維持**無狀態一次性執行**,狀態只存在本機快照檔。「哪些變動值得吵使用者」這種語意判斷留給 Hermes 的 LLM 層,本程式只做確定性的抓取 / 比對 / 輸出。

## Goals / Non-Goals

**Goals:**
- 一支 Go 單一執行檔,子命令 `poll`(抓+比對+印差異+寫快照)、`list`(列當前榜單)與 `get`(依識別碼把磁力連結印到標準輸出)。
- 第一版追蹤類別 `207, 208, 300, 301, 303, 401, 601`,各取 apibay top100 榜單。
- 確定性差異:新進榜、離榜、排名變動、種子數變動。
- 差異以結構化 JSON 印到標準輸出,供 Hermes 的 LLM 層判讀與通知。
- 只把磁力連結字串印到標準輸出;驗證與下載由呼叫者以管道交給 p2pscout,本工具不執行、不相依任何下載程式。

**Non-Goals:**
- 不內建排程 / 常駐(交給 Hermes cron)。
- 不做 LLM 判斷(留 Hermes)。
- 不自己下載 / 不自己搜尋(下載 p2pscout→aria2;搜尋 p2pscout)。
- 不做互動式審批介面(第一版手動 `get <id>`)。
- 不做資料庫(本機 JSON 快照足矣)。

## Decisions

### D1. 資料源:apibay top100 預編檔,逐類別並行抓取
取 `https://apibay.org/precompiled/data_top100_<cat>.json`,七個類別並行。
- 為何:已實測可用、回傳含 `info_hash`+`seeders`+`category`、無需逐筆抓詳情、不靠網頁解析。
- 替代:`q.php` 搜尋端點(適合關鍵字、不適合「榜單」);爬 TPB 網頁(脆弱,否決)。
- 注意:top100 預編檔數值欄位為整數,`q.php` 為字串 —— 用容忍式解碼(`json.Number`)吸收差異。

### D2. 正規化條目與識別碼
`Entry{ InfoHash, Title, Category, Rank, Seeders, Leechers, SizeBytes, Added }`。
對外識別碼(通知與 `get` 用)= `<category>:<infohash前12碼>`,人類好讀又夠唯一。完整 infohash 存在快照供 `get` 還原磁力連結。

### D3. 快照:本機單一 JSON 檔,原子寫入
`Snapshot{ TakenAt, Entries map["<cat>:<infohash>"]Entry }`。預設路徑 `${XDG_STATE_HOME:-~/.local/state}/p2p-ranking-board/snapshot.json`,可用 `--state` 覆寫。寫入先寫 `*.tmp` 再 `rename`(同目錄,原子)。
- 為何 key 用 `cat:infohash`:同一資源可同時上多個類別榜,排名是每類別獨立,複合鍵讓「排名變動」比對落在正確類別。
- 替代:SQLite(過度設計,單檔 JSON 已足);per-category 多檔(增加一致性負擔)。

### D4. 差異引擎(純函式)
輸入「上次 Entries」+「本次 Entries」,輸出 `[]Change{ Type, Entry, PrevRank, PrevSeeders }`。
- 新進榜:本次有、上次無。
- 離榜:上次有、本次無。
- 排名變動:兩次都有、`Rank` 不同。
- 種子數變動:兩次都有、`Seeders` 差異超過設定比例(預設 ±20%,避免抖動噪音)。
- 首次執行(無快照):視本次為基準,輸出空差異。

### D5. 通知:可插拔介面,第一版只實作 stdout
`type Notifier interface { Notify(ctx, []Change) error }`。第一版 `stdoutNotifier` 把差異印成人類可讀文字 + 一份 `--json` 結構化輸出。門檻過濾(`--only=new,rankjump` 等)在送進 Notifier 前先濾。
- 為何:符合「Go 純確定性、判斷留 Hermes」——stdout 的結構化差異正是 Hermes LLM 的輸入。Telegram 等之後加一個 Notifier 實作即可,不動比對邏輯。

### D6. 只輸出磁力連結,以管道串接(零相依)
`get <id>`:從最近快照查出條目 → 組磁力連結 → **印到標準輸出,結束**。本工具不驗證、不下載、不 `exec` 任何程式,也不知道下游是誰。
- **組合方式 = Unix 管道**:`p2p-ranking-board get <id> | p2pscout get --magnet - --auto`。ranking-board 與 p2pscout 互不相依、各自獨立;串接是呼叫者(使用者 / Hermes)的事。
- **為何這樣切**:符合「ranking-board 只提供 magnet/seed」的定位。下載程式(p2pscout)自己管 aria2 設定,ranking-board 完全不碰 aria2,連線/密鑰問題不存在於此。
- 標準輸出 MUST 乾淨(只有磁力連結),診斷訊息走標準錯誤,管道才不會被污染。
- 替代(已否決):由 ranking-board `exec` p2pscout 並轉發 aria2 參數 —— 造成雙向知識與耦合,違反定位。

### D7. 模組與結構
`github.com/freebooters/p2p-ranking-board`,Go 1.25+。`internal/source`(apibay)、`internal/snapshot`、`internal/diff`、`internal/notify`、`internal/resolve`(id→magnet)、`cmd/p2p-ranking-board`(poll/list/get 子命令)。對應五個 spec 能力。

## Risks / Trade-offs

- apibay 不穩 / 地區封鎖 → 任一類別抓取失敗就**不覆寫快照**、非零退出讓 cron 下輪重試;部分類別成功則只比對成功的部分並記錄哪些失敗。
- top100 固定 100 筆/類別 → 榜尾進出的小變動天然被截斷;這是「榜單」語意,可接受,文件註明。
- 數值欄位型別不一(int vs string)→ 容忍式解碼(D1)。
- 通知風暴(首次大量、或榜單洗牌)→ 門檻過濾(D4 種子數比例、D5 `--only`)+ 首次執行不通知(D4)。
- 時鐘 / 重複通知 → 快照持久化保證同狀態不重複通知;cron 重入靠快照天然冪等。

## Migration Plan

新專案,無既有資料。部署 = 編譯出執行檔 + Hermes 設一條 cron 跑 `poll`。回滾 = 移除 cron;快照檔可刪(下次執行重建基準)。

## Open Questions

- 種子數變動門檻預設值(暫定 ±20%)是否合用,待實跑調整。
- 是否要追 leechers 變動(暫定不追,只看 seeders)。
- 通知文字格式細節(留待 stdout Notifier 實作時定)。
