# 永續合約 AI 交易 Dashboard
[![BuyMeACoffee](https://raw.githubusercontent.com/pachadotdev/buymeacoffee-badges/main/bmc-donate-yellow.svg)](https://buymeacoffee.com/aps32777x)

*(English explanation available at the bottom of this file: [English Overview](#english-overview))*

Go + Gin + PostgreSQL 後端、React + Vite 前端的 Binance USDS-M 永續合約中線交易系統。
**串接 Binance 正式環境（mainnet）真實帳戶，會自動下真實市價單**——這跟姊妹專案
`TWSEDailyTrading`（只做模擬下單）不同，請務必讀完「範圍與風險」再啟動。

## 專案結構

```
backend/          Go + Gin API server，internal/ 底下依功能拆套件
  internal/binance/     Binance USDS-M Futures REST + WebSocket 客戶端（無中介bridge，直接串官方API）
  internal/strategy/    技術指標 + 規則型策略（MA/RSI/MACD/VWAP，UTC日界重置）
  internal/ai/          Claude API訊號產生（方法論參考自claude-trading-skills，見下方）
  internal/signalengine/ AI訊號產生+持久化+廣播的共用路徑
  internal/autotrader/  安全關鍵核心：規則轉變偵測→AI確認→保證金×槓桿換算下單量→下真實單→紀錄
  internal/settings/    Kill switch、槓桿、固定保證金金額、保證金模式
  internal/httpapi/     REST API
  internal/positionstore/ 即時持倉/餘額快取（由user data stream即時更新）
frontend/          React + Vite + TypeScript dashboard
docker-compose.yml  Postgres only（沒有中介bridge process，直接連Binance公開API）
```

## 快速開始

這台機器可能同時跑其他專案，所以刻意不用常見的預設 port：

| 服務 | Port |
|---|---|
| Postgres (host) | 55532 |
| Backend API | 8280 |
| Frontend (vite dev) | 5290 |

### 1. 啟動資料庫

```bash
cp .env.example .env
docker compose up -d postgres
```

### 2. 填入真實金鑰（`.env`）

- `BINANCE_API_KEY` / `BINANCE_API_SECRET`：到
  https://www.binance.com/en/my/settings/api-management 建立，**需開啟Futures交易權限**，
  建議加IP白名單。**這是你的真實mainnet帳戶**，伺服器一啟動、只要這兩個key有填、
  `ANTHROPIC_API_KEY`也有填，就會開始自動評估並可能真的下單（見下方「自動交易怎麼運作」）。
- `ANTHROPIC_API_KEY`：https://console.anthropic.com 。沒填的話AI端點回傳
  `{configured:false}`，自動交易引擎也因此形同停用（規則訊號本身永遠不會單獨觸發下單，
  一定要AI確認過）。

### 3. 啟動後端

```bash
cd backend
go run ./cmd/server
```

需要先把根目錄 `.env` 灌進環境變數（`set -a && source ../.env && set +a`，或你習慣的方式）。
後端啟動時會自動跑 DB migration。

### 4. 啟動前端

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

打開 http://localhost:5290。

## 自動交易怎麼運作

每根K線收盤後（間隔由`KLINE_INTERVAL`設定，預設5分鐘；`internal/autotrader.Trader.OnCandleClose`）：

1. 用規則型策略（`internal/strategy.Decide`：MA5/MA20交叉 + VWAP + RSI + MACD柱狀）判斷方向。
2. **每一次評估都會寫進`auto_trade_log`**（包含觀望/略過，不只是真的下單的那幾次）——
   Dashboard最下方「自動交易紀錄」可以看到完整過程與原因，這是刻意的透明度設計。
3. 只有當規則判斷「轉變成」或「翻轉」BUY/SELL（不是每根都still是同方向）才會觸發一次
   Claude呼叫，避免同一個趨勢每根K線都重複花錢問AI。
4. Claude判斷後如果也是BUY/SELL（不是HOLD），才會進到下單流程：
   - Kill switch檢查（dashboard右下角「暫停自動下單」開關）
   - 同方向已有持倉 → 略過，不加碼
   - 冷卻時間（同一合約至少間隔一根K線）與每日下單上限（預設20筆/UTC天）
   - **用交易所真實exchangeInfo，依「保證金×槓桿」換算出的名目金額算出能下的最大量**，
     算不出來（例如BTCUSDT/ETHUSDT的最小下單量本身名目價值就超過目前的保證金×槓桿）
     就略過並記錄原因，絕不會為了成交而超過這個換算出來的名目金額
   - 通過以上全部才會真的呼叫Binance下市價單，隔離保證金（保證金金額與槓桿倍數都可在
     dashboard調整：保證金預設5USDT、槓桿1-25x。**名目倉位=保證金×槓桿，沒有另外的
     絕對上限**——這是使用者的明確選擇：把槓桿調高，倉位就會等比放大，是設計行為
     不是漏洞；單筆風險由固定的保證金金額界定，不是由名目倉位界定）
5. **市價單一旦成交、且是從空手新開倉（不是既有反向倉位的部分減碼），立刻用AI訊號裡
   算好的`stop_loss`/`take_profit`價位掛上停損停利單**（`internal/autotrader.
   placeProtectiveOrders`）——止盈止損在下單當下就設定好，不是等之後某個時間點才補掛，
   也不依賴這個後端程序持續運作才能生效（單子是掛在交易所上的，不是本地邏輯在盯盤）。
   技術上是透過Binance的Algo Order服務（`POST/GET/DELETE /fapi/v1/algoOrder`）下
   `STOP_MARKET`/`TAKE_PROFIT_MARKET`兩張`closePosition=true`的條件單——USDS-M
   Futures已在2025-12-09把這類條件單從舊的`POST /fapi/v1/order`遷移到這個新服務，
   舊路徑現在會直接回傳`-4120`錯誤拒絕。用`closePosition=true`（不指定數量，觸發時
   直接平掉當下的全部倉位）而不是`reduceOnly`+固定數量，一來不用煩惱倉位之後被部分
   減碼導致數量對不上，二來Binance本身保證「同方向同類型最多只能有一張`closePosition`
   單」、且其中一張觸發把倉位打平後，另一張會被交易所自動取消——不需要自己額外寫
   OCO（一邊成交、另一邊自動取消）的邏輯。若停損停利單掛失敗，只會記log、不會讓已經
   成交的進場單回滾（倉位本來就已經在交易所上是真的），下次巡邏（見下方）能透過
   `GET /api/account/orders`發現少了停損停利單。

手動下單（dashboard「手動下單」面板）與平倉按鈕走同一條保證金×槓桿的下單量驗證邏輯，
但手動下單/平倉**不受kill switch影響**（手動操作本身就是明確的override；平倉更是純粹
的風險控管，理應隨時可執行不受任何開關限制）。手動下單目前**不會**自動掛停損停利——
只有自動交易引擎的進場單有AI算好的價位可以用，手動點擊BUY/SELL沒有對應的訊號可參考。

## 一個真實的技術限制：BTC/ETH在小保證金下可能無法下單

Binance合約的最小下單量（LOT_SIZE/MARKET_LOT_SIZE的minQty）與最小名目金額
（MIN_NOTIONAL）依合約而異。實測（開發時抓到的真實資料）：BTCUSDT的MIN_NOTIONAL
本身就是$50、ETHUSDT是$20——如果目前「保證金×槓桿」換算出來的名目金額小於這個門檻
（例如保證金5USDT、槓桿3x＝$15名目，還是不夠BTCUSDT的$50門檻），這兩個合約就無法
自動或手動下單——dashboard會標示「僅看盤」，只顯示K線與AI訊號，不會嘗試下單（也不會
為了硬要成交而偷偷超過換算出來的金額，那樣違反你設定的保證金/槓桿）。把槓桿調夠高
（例如5USDT保證金×10x＝$50名目，剛好貼上BTCUSDT門檻）就能讓BTC/ETH也變得可下單——
這跟直接調高一個固定的名目上限效果一樣，只是透過槓桿間接達成。SOLUSDT/XRPUSDT/
ADAUSDT/DOGEUSDT/TRXUSDT等價格較低的合約門檻低很多，即使在較低槓桿下也通常沒問題。
真正可不可下單一律以`GET /api/watchlist`回傳的即時`tradable_at_cap`欄位為準（現場查
`exchangeInfo`、用當下設定的保證金×槓桿算出來的，不是寫死的清單）。

## AI 方法論來源

`internal/ai/signal.go`的判斷框架（趨勢/拉回結構分類、量價背離判讀、支撐壓力角色互換、
依訊號一致數量校準信心、強制列出反方向風險）參考自
[tradermonty/claude-trading-skills](https://github.com/tradermonty/claude-trading-skills)
的`technical-analyst`/`breakout-trade-planner`等skill方法論後改寫——該repo是給美股
波段交易用的Python skill框架（VCP screener、部位大小計算、交易日誌等），跟本專案的
Go後端/永續合約5分K短中線場景不相容，沒有直接安裝或呼叫，只是把裡面可轉移的技術分析
原則（跨資產、跨時間週期通用）改寫進本專案自己的prompt文字，並加上永續合約特有的
資金費率(funding rate)脈絡——這是股票沒有的訊號，正費率代表多方擁擠、持倉成本隨時間
增加，已整合進prompt要求AI在rationale中明確說明資金費率是否支持當下判斷方向。

## 範圍與限制（刻意的簡化，不是漏做）

- 沒有回測引擎、沒有AI每日選股——這兩個功能在姊妹專案`TWSEDailyTrading`裡，本專案沒有
  被要求要有，刻意不做以控制範圍。
- 單一帳戶，沒有多使用者登入系統。
- One-way position mode（不是hedge mode），同一合約不會同時持有多空兩個方向的倉位。
- 反手訊號（例如目前空單，AI判斷該做多）：新單一樣照「保證金×槓桿」的固定金額計算，
  用Binance的自然netting機制部分減碼原本的空單，不會為了「反手」而下一筆更大的單去
  蓋過既有倉位。

## 開發時我做過的驗證

- `go build ./...` / `go vet ./...` / `go test ./...` 全過，包含`internal/strategy`
  的規則判斷與UTC日界VWAP重置測試、`internal/binance`的`MaxQtyForCap`安全下單量計算
  測試（针對BTCUSDT/ETHUSDT驗證「最小下單量都超過$10所以標記不可下單」、對
  SOLUSDT/XRPUSDT/ADAUSDT/DOGEUSDT驗證「能算出$10以內的合法下單量」，兩種情況都跟開發
  時抓到的Binance真實`exchangeInfo`數字對過）。
- 前端 `tsc -b` 型別檢查 / `vite build` 全過。
- **實際跑過整條路徑**（原始版本，當時`KLINE_INTERVAL=15m`）：`docker compose up -d
  postgres` + `go run` 後端（真實mainnet API key）+ `npm run dev` 前端，瀏覽器實測確認：
  8檔關注清單都能即時串流K線並正確顯示K線圖、`/api/watchlist`的`tradable_at_cap`欄位
  跟開發時期算的BTC/ETH不可下單、SOL($9.36)/XRP($9.99)/ADA($9.88)/DOGE($9.96)可下單
  完全吻合、帳戶餘額面板正確顯示真實USDT餘額、kill switch正確顯示「運作中（LIVE）」、
  手動點擊「產生訊號」成功呼叫真實Claude API並回傳結構化訊號（HOLD、信心30%、進場/
  停損/停利價位、內含資金費率分析的繁中rationale）。**沒有實際點擊BUY/SELL或平倉
  按鈕**——那會動用真實資金，刻意留給你自己確認後操作，不是遺漏。
- 之後應要求改成`KLINE_INTERVAL=5m`（預設值）、槓桿上限從5x放寬到25x（$10名目倉位
  上限本身沒變，仍是伺服器端寫死的絕對上限——槓桿只影響保證金效率與強平距離，不影響
  單筆最大虧損），並把`candles_15m`表改名為`candles`（interval-agnostic，因為間隔已
  可透過設定調整；改名的migration同時truncate了舊表，因為15分K與5分K bar混在同一個
  序列裡會讓SMA/RSI/MACD這類假設固定間隔的指標算錯）。已跑過`go build`/`go vet`/
  `go test`與前端`tsc -b`/`vite build`全過，並重新瀏覽器實測確認5分K圖表正確渲染、
  槓桿輸入框上限確實變成25。
- **發現並修好一個嚴重的環境層級bug：長連線WSS在這台機器上完全收不到資料**。用
  Claude Code的Monitor工具直接開一條乾淨的WebSocket連線到Binance公開的combined
  stream，handshake成功但30秒內一個frame都沒收到——這不是我們程式碼的bug，是這台機器
  的網路環境（防火牆/VPN/防毒軟體其中之一）會讓長連線WSS的handshake過，但之後的資料
  frame被靜默擋掉，plain HTTPS REST完全不受影響。這解釋了兩個現象：(1) `/api/market/
  funding`看起來「即時更新」其實是因為WS快取永遠是空的，每次都走進REST fallback分支，
  不是真的WS在動；(2) `candles`表在SeedHistory第一次寫入後就再也沒更新過——因為唯一
  會觸發`OnCandleClose`（驅動整個規則→AI→下單流程的入口）的路徑只有WS，這條路徑
  死掉代表**自動交易引擎從一開始就沒有真正跑過**。修法：在`internal/marketdata`加了
  `PollCandles`/`PollFunding`兩個REST輪詢的備援迴圈（`cmd/server/streams.go`裡跟WS
  串流用同一個`runCtx`生命週期），K線輪詢每8秒抓最近2根K線（已收盤的+正在形成的），
  用「這根收盤K線的時間戳是不是這個合約第一次看到」來可靠判斷收盤事件並觸發
  `OnCandleClose`（REST回傳的收盤K線本來就是確定性的，不像WS的`x:true`旗標可能根本
  收不到）；同時幫WS路徑加了30秒無資料自動判定死線重連（`internal/binance/marketws.go`
  的`readIdleTimeout`），並在`internal/autotrader`加了`SyncAccountState`/
  `ReconcilePendingOrders`兩個REST備援（每15秒跑一次，`cmd/server/main.go`的
  `periodicAccountSync`），確保就算user data stream的WS也一樣收不到資料，持倉/訂單
  狀態依然會透過REST正確更新——這對一個真的在下真錢單的bot來說是正確性關鍵，不只是
  「圖表看起來卡住」的表面問題。**修好後已實測確認**：`candles`表的K線時間戳開始正常
  往前推進、後端log出現真實的`rule trigger ... requesting AI evaluation`（BTCUSDT/
  ETHUSDT/DOGEUSDT三個合約在同一次收盤各自觸發了一次真實Claude呼叫）、
  `auto_trade_log`正確記錄了規則判斷、AI最終決定（含兩次「ai_overrode_rule_to_hold」
  ——規則判斷SELL但AI判斷觀望，正確地沒有下單）、dashboard的「自動交易紀錄」面板即時
  顯示這些紀錄。
- 同時修好一個前端問題：K線圖表沒有針對合約價位動態調整小數位精度，預設2位小數對
  BTCUSDT（~$79000）沒問題，但DOGEUSDT(~$0.09)/ADAUSDT(~$0.22)/1000PEPEUSDT(~$0.004)
  這類低價合約會被四捨五入到看起來完全沒在動——這正是使用者回報「盤面沒有在動」的
  另一半原因（另一半是上面的WS bug本身）。`CandleChart.tsx`現在依收盤價量級動態決定
  精度（2到8位小數）。已實測確認DOGEUSDT圖表切換後Y軸正確顯示5位小數且K棒清楚可見
  真實價格波動。
- 使用者也直接問過「25x槓桿是不是能讓BTC/ETH可以下單了」——答案是不行：$10上限管的
  是**名目倉位價值**（qty×price），BTCUSDT/ETHUSDT交易所允許的最小下單量本身換算
  出來的名目價值（$50/$20）就已經超過$10，這件事跟槓桿倍數完全無關——槓桿只影響
  「這$10的倉位要佔用多少保證金」，不影響交易所允許的最小下單門檻。已用當下即時
  `/api/watchlist`資料重新確認：BTCUSDT/ETHUSDT在25x槓桿下`tradable_at_cap`依然是
  `false`，SOL/XRP/ADA/DOGE/TRX依然是`true`，數字跟先前一致。
- **改成保證金固定制**：使用者要求把下單邏輯從「名目倉位固定$10上限」改成「每筆保證金
  固定5USDT，名目倉位=保證金×槓桿」。問過使用者要不要在這之上額外加一個絕對名目上限
  （避免以後調高槓桿時倉位無限放大），使用者明確選擇不要，完全依保證金×槓桿計算——
  已照這個決定實作，`internal/settings.Settings`把`NotionalCapUSD`（含$10硬上限常數）
  整個換成`MarginUSD`（固定保證金，1-50 USDT範圍只是防打字錯誤的寬鬆檢查，不是行為
  限制）+ 新增`EffectiveNotionalUSD`（=MarginUSD×Leverage，唯讀，方便前端顯示換算後的
  金額），DB migration把`settings.notional_cap_usd`欄位rename成`margin_usd`並把預設值
  跟現有那筆設定都改成5.00。所有原本吃`NotionalCapUSD`的地方（`internal/autotrader`的
  自動/手動下單、`GET /api/watchlist`的`tradable_at_cap`可行性計算）全部改吃
  `EffectiveNotionalUSD`。Dashboard的「自動交易」面板現在有兩個輸入框（保證金USDT+槓桿），
  旁邊即時顯示換算後的名目金額；「手動下單」與「僅看盤」的說明文字也都改成
  動態顯示目前的保證金/槓桿/換算金額，不再寫死$10。已跑過`go build`/`go vet`/
  `go test`/前端`tsc -b`/`vite build`全過。
- **自動進場時同步掛停損停利**：使用者要求「止盈損要下單時就設定好」。實作前先查證了
  一個重要的API變更——USDS-M Futures已在2025-12-09把`STOP_MARKET`/`TAKE_PROFIT_MARKET`
  等條件單從舊的`POST /fapi/v1/order`遷移到新的Algo Order服務，舊路徑現在回傳`-4120`
  拒絕（用WebSearch/WebFetch對照官方文件與開發者社群討論查證過，包括確認新端點的
  正確參數是`triggerPrice`不是`stopPrice`、`POST/GET/DELETE /fapi/v1/algoOrder`三個
  端點、回傳用`algoId`/`algoStatus`而不是`orderId`/`status`，以及`closePosition=true`
  的兩張單一邊觸發後Binance會自動取消另一邊，不需要自己實作OCO邏輯——這幾點都是這次
  改動能不能安全動用真錢的關鍵，沒有直接假設就先查證過）。新增`internal/binance/
  algo.go`（`PlaceClosePositionAlgoOrder`/`QueryAlgoOrder`/`CancelAlgoOrder`）、
  DB migration把`orders`表加上`algo_id`欄位（跟`binance_order_id`是兩個獨立的ID
  空間，一張單只會用到其中一個）、`internal/autotrader.execute()`在**從空手新開倉**
  成交後（不是既有反向倉位的部分減碼）呼叫新的`placeProtectiveOrders`掛好停損停利、
  `ReconcilePendingOrders`也擴充成同時巡邏一般單與algo單兩種狀態。已跑過`go build`/
  `go vet`/`go test`全過。**還沒觀察到真實觸發**（需要等下一次全新進場才會掛
  出真的停損停利單，我沒有自己額外對使用者既有的TRXUSDT倉位下單來測試，因為那會是
  我自己選一個價位替使用者的真實倉位掛真錢條件單，不是單純驗證）——建議之後留意
  `GET /api/account/orders`裡`order_type`為`STOP_MARKET`/`TAKE_PROFIT_MARKET`的
  紀錄，確認第一次真實觸發時行為符合預期。

---

## English Overview

*This section is a standalone English explanation of the system as it exists today. It is not a line-by-line translation of the Chinese sections above (which include an append-only development log written during earlier build sessions) — some features described here (the Silver Bullet strategy, the backtest optimizer, and AI daily watchlist selection) were added after most of the Chinese content was written and aren't reflected there yet.*

### What this is

A Go + Gin + PostgreSQL backend with a React + Vite + TypeScript frontend for **automated perpetual-futures trading on Binance USDS-M**, with Claude (Anthropic's API) as a second-opinion confirmation layer before any real order is placed.

**This connects to your real Binance mainnet account and places real market orders.** Unlike the sibling project `TWSEDailyTrading` (which only simulates orders), this one trades real money the moment the backend starts, provided `BINANCE_API_KEY`/`BINANCE_API_SECRET` and `ANTHROPIC_API_KEY` are all set. Read this whole section, especially "Known limitations", before running it.

### Project structure

```text
backend/          Go + Gin API server, organized by package under internal/
  internal/binance/       Binance USDS-M Futures REST + WebSocket client (talks to Binance's public API directly, no bridge process)
  internal/strategy/      The Silver Bullet setup detector (see below) - the local rule that gates the AI confirmation step
  internal/backtest/      Replays historical candles through the detector to simulate trades and grid-search parameters
  internal/ai/            Claude API calls: trade-signal confirmation and daily watchlist selection
  internal/watchlistai/   Candidate sourcing + signal-frequency backtesting for the AI daily watchlist pick
  internal/signalengine/  Shared path for generating + persisting + broadcasting an AI trade signal
  internal/autotrader/    Safety-critical core: rule detection -> AI confirmation -> margin*leverage sizing -> real order -> protective stop/take-profit -> logging
  internal/settings/      Kill switch, leverage, fixed margin per order, margin mode
  internal/httpapi/       REST API
  internal/positionstore/ Live position/balance cache, updated from Binance's user-data-stream
frontend/          React + Vite + TypeScript dashboard
docker-compose.yml  Postgres + backend + frontend - no bridge process, the backend container talks to Binance's public API directly
```

### Quick start

This machine may run other projects side by side, so the default ports are deliberately non-standard:

| Service              | Port  |
| -------------------- | ----- |
| Postgres (host)      | 55532 |
| Backend API           | 8280 |
| Frontend (vite dev)  | 5290  |

**Option A - Docker Compose (whole stack):**

1. `cp .env.example .env` and `cp .env.prompts.example .env.prompts`, then fill in real credentials in `.env` (see step 2 below for which ones matter).
2. `docker compose up -d --build` - builds the backend image, runs Postgres/backend/frontend together, migrations run automatically on backend startup. Open http://localhost:5290.
3. Note: the backend container does NOT read `.env.prompts` (see "AI prompts" below) - it always uses the built-in default prompts. Use Option B if you need the customized prompt text.

**Option B - native processes (what the detached-deployment tooling in this repo assumes):**

1. `cp .env.example .env` then `docker compose up -d postgres` (just the database).
2. **Fill in real credentials in `.env`**: `BINANCE_API_KEY`/`BINANCE_API_SECRET` (create at binance.com/en/my/settings/api-management with Futures trading enabled, IP-whitelisting recommended - **this is your real mainnet account**) and `ANTHROPIC_API_KEY` (console.anthropic.com - without this, the AI endpoints report `{configured:false}` and auto-trading is effectively a no-op, since the rule alone never places an order by itself).
3. **Backend**: `cd backend && go build -o cryptotrading-server.exe ./cmd/server && powershell -File .\restart-detached.ps1` (Windows; runs it detached in the background, reading both `.env` and the optional `.env.prompts`) - or just `go run ./cmd/server` in a foreground terminal with `../.env` loaded into the environment. Migrations run automatically on startup.
4. **Frontend**: `cd frontend && cp .env.example .env && npm install && npm run dev`, then open http://localhost:5290.

### Strategy: ICT Silver Bullet (with a 2026-era upgrade)

The local rule (`internal/strategy.DecideSilverBullet`) implements the ICT ("Inner Circle Trader") Silver Bullet setup, extended with several additional confluence checks beyond the classic bare version:

1. **Liquidity sweep** - price wicks beyond a recent swing high/low, then closes back inside (a stop-hunt).
2. **Displacement-quality Fair Value Gap** - the 3-candle imbalance following the sweep must be formed by a genuine "displacement" candle (large body, small opposing wick), filtering out weak/noisy gaps.
3. **Optimal Trade Entry (OTE)** - rather than entering immediately when the gap confirms, the setup waits (up to a configurable number of bars) for price to retrace into the 62%-79% Fibonacci zone of the sweep-to-displacement leg before triggering entry.
4. **Breaker Block confluence (Unicorn Model)** - the OTE zone must overlap a nearby Breaker Block (the order block the sweep broke through).
5. **SMT divergence** - a correlated anchor symbol (BTCUSDT, or ETHUSDT when the symbol itself is BTCUSDT) must *not* confirm the same extreme, indicating the sweep is specific to this symbol rather than a broad-market move.

Unlike ICT's original session-gated definition (the classic NY 10:00-11:00 / 14:00-15:00 windows), this detector does **not** gate on time-of-day - crypto perpetuals trade 24/7, so a qualifying setup is evaluated whenever it occurs. All five conditions are checked mechanically before Claude ever sees the setup; the AI's job is judging how convincing the setup is, not re-deriving it. The system prompt Claude receives is intentionally short and framework-agnostic (see "AI prompts" below) - it doesn't name or explain ICT concepts, it just reviews whatever the rule engine already detected.

Because BTC/ETH can be dropped from the tradable watchlist by the AI daily selection (see below) while still being needed as the SMT anchor, the market-data stream always keeps their candles fresh regardless of the trading watchlist - a `Trader.isWatchlisted` guard makes sure they're never auto-traded unless they're genuinely on the watchlist too.

### How auto-trading works

On every candle close (`internal/autotrader.Trader.OnCandleClose`):

1. Evaluate the Silver Bullet rule above. Every evaluation is written to `auto_trade_log` (including HOLDs, not just executed trades) for full transparency - visible in the dashboard's "Auto-Trade Log" panel.
2. Only a genuinely new setup (deduped by the Fair Value Gap's bar timestamp) triggers a Claude API call - HOLD never calls the AI, and neither does a setup that was already acted on.
3. If Claude also confirms BUY/SELL (not HOLD), the order pipeline runs: kill-switch check, "already holding this direction" skip, per-symbol cooldown and a daily order cap, and order sizing computed from Binance's real exchange filters against the **margin x leverage** notional (adjustable in the dashboard - fixed margin per order, e.g. 5 USDT, x leverage 1-25x; there is deliberately no separate absolute notional ceiling on top of that, by explicit user choice).
4. On a successful fresh entry (not a partial reduction of an existing opposite position), stop-loss and take-profit orders are placed immediately using the price levels Claude's signal computed, via Binance's Algo Order service (`STOP_MARKET`/`TAKE_PROFIT_MARKET`, `closePosition=true`) - these live on the exchange and don't depend on this backend process staying up.

Manual order placement and flatten (dashboard) share the same margin x leverage sizing/validation logic, but bypass the kill switch (manual action is itself an explicit override). Manual orders do not get automatic stop-loss/take-profit - only AI-confirmed automatic entries have signal-computed price levels to use.

### AI daily watchlist selection

`internal/watchlistai` can replace the trading watchlist once a day (or on demand via `POST /api/watchlist/ai-refresh`): it pulls the full USDT-margined perpetual universe, ranks by liquidity and $-cap feasibility, then - for each candidate - actually **backtests the currently-live Silver Bullet rules** against its recent history (`AnnotateSignalFrequency`) to count how many real signals they would have produced. Claude then picks the final watchlist symbols primarily by that empirical signal count, not by guessing from volatility or 24h price change. Every run is recorded (`ai_watchlist_runs`/`ai_watchlist_picks`) for audit.

### Backtest optimizer

`internal/backtest` replays historical candles through the Silver Bullet detector to simulate trades, then `POST /api/strategy/optimize` grid-searches the five tunable parameters (swing lookback, FVG size threshold, sweep window, stop buffer, risk:reward) over ~60 days of history, split into a training period and a held-out validation period. Candidates are ranked by **validation-period** Sharpe ratio (never training Sharpe) and must clear a minimum trade-count/Sharpe bar to be eligible - the actual defense against overfitting, since only out-of-sample performance decides the winner. The ICT-2026 structural thresholds (displacement body %, OTE retracement bounds, etc.) are held fixed rather than grid-searched, since they're meant to encode established rules, not free parameters to curve-fit.

### AI prompts

The two system prompts Claude receives (`internal/ai.GenerateSignal` and `SelectDailyWatchlist`) are overridable via `AI_SIGNAL_SYSTEM_PROMPT`/`AI_WATCHLIST_SYSTEM_PROMPT` in a separate `.env.prompts` file (copy `.env.prompts.example` to get started), using a multi-line heredoc syntax (`KEY<<EOF` ... a line that is exactly `EOF` ends it). This lives outside `.env` itself because Docker Compose's own `.env` parser fails hard on that syntax - keeping `.env` plain `KEY=value` is what makes `docker compose up` work at all. Only the native restart path (`restart-detached.ps1`, or manually loading both files into the environment before `go run`) reads `.env.prompts`; the docker-compose `backend` service does not, and always uses the built-in default prompts in `internal/config/config.go`. Either way, editing the prompt text tunes the AI's wording/judgment without a Go rebuild - a backend restart is still required to pick up a change (read once at startup, not live-reloaded).

### Known limitations (deliberate scope choices, not oversights)

- Single account, no multi-user login system.
- One-way position mode (not hedge mode) - a symbol never holds both a long and a short position at once.
- A reversal signal (e.g. currently short, AI now says go long) sizes the new order the same fixed margin x leverage way and relies on Binance's natural netting to partially close the existing position - it never places an oversized order specifically to "flip" a position in one shot.
- BTCUSDT/ETHUSDT may show as "chart only, not tradable" at low margin x leverage settings, since their exchange-minimum order size's notional value ($50/$20 respectively, at time of writing) can exceed a small effective notional (margin x leverage) - this is a real exchange constraint, not a bug; raising leverage (which raises the effective notional) is what makes them tradable again. `GET /api/watchlist`'s `tradable_at_cap` field always reflects live `exchangeInfo` data, never a hardcoded list.
