# change-detection Specification

## Purpose
TBD - created by archiving change add-tpb-ranking-tracker. Update Purpose after archive.
## Requirements
### Requirement: 持久化榜單快照

系統 SHALL 把每次成功抓取的正規化榜單持久化為本機快照,並在下次執行時載入上一次快照作為比對基準。寫入 MUST 為原子性(先寫暫存檔再更名),避免中途中斷導致快照毀損。

#### Scenario: 首次執行無既有快照
- **WHEN** 系統執行時找不到既有快照
- **THEN** 系統 SHALL 把本次榜單視為基準寫入快照,且不產生任何「變動」

#### Scenario: 原子性寫回
- **WHEN** 系統完成本次比對後寫回新快照
- **THEN** 系統先寫入暫存檔再原子性更名為正式快照檔

### Requirement: 比對榜單差異

系統 SHALL 以資訊雜湊為鍵,比對本次榜單與上一次快照,辨識下列變動類型:新進榜、離榜、排名升降、種子節點數變動。

#### Scenario: 偵測新進榜
- **WHEN** 某條目的資訊雜湊出現在本次榜單但不在上一次快照
- **THEN** 系統 SHALL 將其標記為「新進榜」變動

#### Scenario: 偵測排名變動
- **WHEN** 某條目同時存在於兩次榜單但排名不同
- **THEN** 系統 SHALL 將其標記為「排名變動」,並記錄前後名次

#### Scenario: 無變動
- **WHEN** 本次榜單與上一次快照在比對欄位上完全相同
- **THEN** 系統 SHALL 回報空的變動集合

