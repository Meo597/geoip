# GeoIP

本仓库提供基于 `IPInfo Lite` 构建并额外修正的 GeoIP 数据，适合直接用于 `geoip.dat`、`mmdb`、`srs`、`mrs`、Clash、Surge 等路由场景。

## 更新说明

- 每天北京时间 `02:00` 开始更新。
- 受 GitHub Actions 调度、下载源可用性和任务执行时长影响，通常会有几个小时延迟。
- 如果你希望拿到当天相对稳定的一版，通常北京时间 `06:00` 左右更稳。

## 数据基线

- 全部国家和地区基础数据基于 `IPInfo Lite`。
- 扩展 ASN 类别同样基于 `IPInfo Lite` 的 ASN 数据生成。

## 中国大陆特殊处理

- `geoip:cn` 不是直接原样使用 `IPInfo Lite`。
- 中国大陆 `IPv4` 会先从基础库中移除，再使用多个中国大陆 IPv4 列表合并重建：
  - `metowolf/iplist`
  - `zhufengme/block_cn_files`
  - `misakaio/chnroutes2`
  - `gaoyifan/china-operator-ip` 的 `china.txt`
- 中国大陆 `IPv6` 保留 `IPInfo Lite` 基线，同时额外合并 `gaoyifan/china-operator-ip` 的 `china6.txt`。
- 因此当前 `geoip:cn` 的结果，本质上是 `IPInfo Lite` 基线加上针对中国大陆的定制修正。
- 这么做的目的，是让 `geoip:cn` 更偏向路由实用性而不是只看“当前是否对外广播”：即便某些大陆地址段暂时未广播、上游更新滞后，或属于运营商海外骨干与过境网络，也尽量纳入 `geoip:cn`，以减少 `CN` 优先场景下的漏判、绕路和误走代理。

## 附加列表

除常规国家/地区分类外，当前配置还额外提供这些列表：

- `geoip:anycast`（`GEOIP,ANYCAST`）
- `geoip:cloudflare`（`GEOIP,CLOUDFLARE`）
- `geoip:cloudfront`（`GEOIP,CLOUDFRONT`）
- `geoip:facebook`（`GEOIP,FACEBOOK`）
- `geoip:fastly`（`GEOIP,FASTLY`）
- `geoip:google`（`GEOIP,GOOGLE`）
- `geoip:netflix`（`GEOIP,NETFLIX`）
- `geoip:telegram`（`GEOIP,TELEGRAM`）
- `geoip:twitter`（`GEOIP,TWITTER`）
- `geoip:tor`（`GEOIP,TOR`）

这些列表的数据来源包括官方 IP 列表、公开前缀列表，以及基于 `IPInfo Lite` ASN 数据提取的结果。

## 下载地址

- `GitHub Releases`
  - `https://github.com/Meo597/geoip/releases/latest`
- `release` 分支
  - `https://github.com/Meo597/geoip/tree/release`
- Raw 直链模板
  - `https://raw.githubusercontent.com/Meo597/geoip/release/<path>`
- jsDelivr 直链模板
  - `https://cdn.jsdelivr.net/gh/Meo597/geoip@release/<path>`

常用文件：

- `https://raw.githubusercontent.com/Meo597/geoip/release/geoip.dat`
- `https://raw.githubusercontent.com/Meo597/geoip/release/geoip-only-cn-private.dat`
- `https://raw.githubusercontent.com/Meo597/geoip/release/geoip-asn.dat`
- `https://raw.githubusercontent.com/Meo597/geoip/release/Country.mmdb`
- `https://raw.githubusercontent.com/Meo597/geoip/release/Country-without-asn.mmdb`
- `https://raw.githubusercontent.com/Meo597/geoip/release/Country-only-cn-private.mmdb`
- `https://raw.githubusercontent.com/Meo597/geoip/release/Country-asn.mmdb`

按格式拆分的目录：

- `https://github.com/Meo597/geoip/tree/release/dat`
- `https://github.com/Meo597/geoip/tree/release/srs`
- `https://github.com/Meo597/geoip/tree/release/mrs`
- `https://github.com/Meo597/geoip/tree/release/clash`
- `https://github.com/Meo597/geoip/tree/release/surge`
- `https://github.com/Meo597/geoip/tree/release/text`
- `https://github.com/Meo597/geoip/tree/release/nginx`

## License

- [LICENSE](./LICENSE)
- [LICENSE-GPL](./LICENSE-GPL)

## 致谢

本项目从 [`Loyalsoldier/geoip@b84c0db4`](https://github.com/Loyalsoldier/geoip/commit/b84c0db4ad7286eb5cec870cabf6642a156290bc) 分叉而来。
