# 订阅价格展示

入口：**系统设置 → 订阅价格**。此功能展示 OpenPrice 收录的 App Store 订阅价格，不参与站内 API 计费、用户余额或支付。

## 启用与使用

1. 打开「启用订阅价格」，选择刷新周期：5 小时、18 小时、每天或每周。
2. 选择展示的产品，以及最多 8 个首页代表地区。中国大陆和最低价地区自动保留，不占代表地区名额。
3. 需要在首页展示时，再打开「在首页展示」，点击「保存价格设置」。首次启用自动同步，也可手动点击「立即刷新」。
4. 首页按产品展示所选套餐的最低价和地区对比；点击价格进入 `/official-prices/chatgpt`、`/official-prices/claude` 或 `/official-prices/grok`，查看全部地区，支持套餐切换、搜索和价格排序。

总开关默认关闭。单独关闭首页展示不会关闭详情页；关闭总开关会停止自动抓取并关闭所有公开价格展示。刷新已打开的页面即可应用新配置。

## 数据口径

- 同一个产品、同一个套餐进行比较，不将年付、月付或不同权益混在一起。
- 首页的最低价基于该套餐全部已收录地区，代表地区选择不会影响最低价结果。
- 来源没有中国大陆报价时明确显示「暂无报价」，不使用其他地区代替。此处展示的是来源收录范围，不保证覆盖全球所有可购买地区。
- 保留原币价格、来源换算的人民币价格、每条报价的来源更新时间和本站同步时间；同步时间不代表报价刚刚变化。
- 页面保留 OpenPrice 来源链接。价格可能变化，实际扣款以 Apple 账单为准。

## 运行与维护

配置保存在现有 settings 存储中。最近一次完整价格快照保存在 `pricing.data_dir/official_subscription_prices.json`（默认 `./data`）。Docker 需保留项目已有的 `/app/data` 持久化挂载，重建容器后可继续读取快照。

后台每分钟检查是否到达刷新时间。三个来源页面均成功解析后才替换快照；网络超时、源站格式变化或写盘失败会保留上次有效报价，后台显示失败状态，按至少 15 分钟间隔重试。若首次同步失败，首页价格板块保持隐藏，详情页提示暂无数据。

抓取只访问固定的 OpenPrice 公开网址，不执行第三方 JavaScript，不增加前端依赖。普通页面访问读取本地快照，不会触发抓取。

## 验证

后端测试：`cd backend && go test ./internal/service -run TestOfficialPrices`

需要联网验收时：`OFFICIAL_PRICES_LIVE_TEST=1 go test ./internal/service -run TestOfficialPricesLiveSource -count=1 -v`。该检查需要运行环境能访问 OpenPrice。

前端测试：`cd frontend && pnpm exec vitest run src/utils/__tests__/officialPrices.spec.ts src/views/__tests__/OfficialPrices.spec.ts`
