# GeoIP

本仓库提供基于 IPInfo Lite 构建并额外修正的 GeoIP 数据，适合直接用于 Xray、Sing-Box、Clash、Surge 等应用。

## 更新说明

数据每天北京时间 02:00 自动更新。受 GitHub Actions 调度延迟影响，工作流实际开始时间可能比计划时间晚几个小时。如果希望比较及时地获取当日更新，建议在北京时间 06:00 左右再下载，太早下载可能仍是前一天的数据。

## 数据基线

全部国家和地区的基础数据基于 IPInfo Lite。除国家和地区外的附加类别主要基于 IPInfo Lite 的 ASN 数据生成，部分也会结合上游直接提供的数据。

## 中国大陆特殊处理

`geoip:cn` 不是直接原样使用 IPInfo Lite。中国大陆 IPv4 会先从基础库中移除，再使用以下多个中国大陆 IPv4 列表合并重建：

- metowolf/iplist
- zhufengme/block_cn_files
- misakaio/chnroutes2
- gaoyifan/china-operator-ip 的 china.txt

中国大陆 IPv6 保留 IPInfo Lite 基线，同时额外合并 gaoyifan/china-operator-ip 的 china6.txt。

因此，当前 `geoip:cn` 的结果，本质上是 IPInfo Lite 基线加上针对中国大陆的定制修正。中国 ISP 的海外骨干路由也会纳入 `geoip:cn`，这类地址通常不直接对外提供服务，单独剔除维护成本较高，实际收益有限，因此保持归入 `geoip:cn`。与此同时，未广播的中国 IP 段同样会纳入 `geoip:cn`，以避免因更新滞后错过可用时机，让这部分地址在开始广播后能够尽快投入使用。

## 附加类别

除常规国家和地区分类外，还额外提供这些类别：

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

这些类别的数据来源包括官方 IP 列表、公开前缀列表，以及基于 IPInfo Lite ASN 数据提取的结果。

## 下载地址

### GitHub Releases

<https://github.com/Meo597/geoip/releases/latest>

### “release” 分支

<https://github.com/Meo597/geoip/tree/release>

- GitHub Raw：`https://raw.githubusercontent.com/Meo597/geoip/release/<path>`
- jsDelivr Cloudflare：`https://cdn.jsdelivr.net/gh/Meo597/geoip@release/<path>`
- jsDelivr Fastly：`https://fastly.jsdelivr.net/gh/Meo597/geoip@release/<path>`

常用文件：

- <https://raw.githubusercontent.com/Meo597/geoip/release/geoip.dat>
- <https://raw.githubusercontent.com/Meo597/geoip/release/geoip-only-cn-private.dat>
- <https://raw.githubusercontent.com/Meo597/geoip/release/geoip-asn.dat>

以下目录中，每个分类对应一个独立文件：

- <https://github.com/Meo597/geoip/tree/release/dat>
- <https://github.com/Meo597/geoip/tree/release/srs>
- <https://github.com/Meo597/geoip/tree/release/mrs>
- <https://github.com/Meo597/geoip/tree/release/clash>
- <https://github.com/Meo597/geoip/tree/release/surge>
- <https://github.com/Meo597/geoip/tree/release/text>
- <https://github.com/Meo597/geoip/tree/release/nginx>

## License

- [CC-BY-SA-4.0](./LICENSE)
- [GPL-3.0](./LICENSE-GPL)

## 致谢

- 本项目从 [`Loyalsoldier/geoip@b84c0db4`](https://github.com/Loyalsoldier/geoip/commit/b84c0db4ad7286eb5cec870cabf6642a156290bc) 分叉而来
- [IPInfo Lite](https://ipinfo.io/lite)
- [metowolf/iplist](https://github.com/metowolf/iplist)
- [zhufengme/block_cn_files](https://github.com/zhufengme/block_cn_files)
- [misakaio/chnroutes2](https://github.com/misakaio/chnroutes2)
- [gaoyifan/china-operator-ip](https://github.com/gaoyifan/china-operator-ip)
