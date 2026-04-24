# GeoIP

本项目是 [Loyalsoldier/geoip](https://github.com/Loyalsoldier/geoip) 的分叉版本。基础功能、输出格式、使用方式和上游更新说明请直接参考上游项目。

This project is a fork of [Loyalsoldier/geoip](https://github.com/Loyalsoldier/geoip). Please refer to the upstream project for the base features, output formats, usage, and release notes.

## 下载地址 / Downloads

构建产物会发布到本仓库的 `release` 分支，并同步上传到 GitHub Releases。

Build artifacts are published to this repository's `release` branch and also uploaded to GitHub Releases.

- GitHub Releases: [https://github.com/Meo597/geoip/releases/latest](https://github.com/Meo597/geoip/releases/latest)
- release 分支 / release branch: [https://github.com/Meo597/geoip/tree/release](https://github.com/Meo597/geoip/tree/release)
- Raw 地址 / Raw URL: `https://raw.githubusercontent.com/Meo597/geoip/release/<FILE>`
- CDN 地址 / CDN URL: `https://cdn.jsdelivr.net/gh/Meo597/geoip@release/<FILE>`

常用文件 / Common files:

| File | Raw | CDN |
| --- | --- | --- |
| `geoip.dat` | [raw](https://raw.githubusercontent.com/Meo597/geoip/release/geoip.dat) | [cdn](https://cdn.jsdelivr.net/gh/Meo597/geoip@release/geoip.dat) |
| `geoip-asn.dat` | [raw](https://raw.githubusercontent.com/Meo597/geoip/release/geoip-asn.dat) | [cdn](https://cdn.jsdelivr.net/gh/Meo597/geoip@release/geoip-asn.dat) |
| `Country.mmdb` | [raw](https://raw.githubusercontent.com/Meo597/geoip/release/Country.mmdb) | [cdn](https://cdn.jsdelivr.net/gh/Meo597/geoip@release/Country.mmdb) |
| `Country-without-asn.mmdb` | [raw](https://raw.githubusercontent.com/Meo597/geoip/release/Country-without-asn.mmdb) | [cdn](https://cdn.jsdelivr.net/gh/Meo597/geoip@release/Country-without-asn.mmdb) |
| `Country-only-cn-private.mmdb` | [raw](https://raw.githubusercontent.com/Meo597/geoip/release/Country-only-cn-private.mmdb) | [cdn](https://cdn.jsdelivr.net/gh/Meo597/geoip@release/Country-only-cn-private.mmdb) |
| `Country-asn.mmdb` | [raw](https://raw.githubusercontent.com/Meo597/geoip/release/Country-asn.mmdb) | [cdn](https://cdn.jsdelivr.net/gh/Meo597/geoip@release/Country-asn.mmdb) |

对应的校验文件在文件名后追加 `.sha256sum`。分列表文件位于 `release` 分支的对应目录，例如 `dat/anycast.dat`、`srs/anycast.srs`、`mrs/anycast.mrs`、`clash/ipcidr/anycast.txt`、`clash/classical/anycast.txt` 和 `surge/anycast.txt`。

Checksum files use the same path with `.sha256sum` appended. Per-list files are available under the corresponding directories in the `release` branch, for example `dat/anycast.dat`, `srs/anycast.srs`, `mrs/anycast.mrs`, `clash/ipcidr/anycast.txt`, `clash/classical/anycast.txt`, and `surge/anycast.txt`.

## 分类 / Categories

国家/地区分类使用两位 ISO 3166-1 alpha-2 代码，例如 `geoip:cn`、`geoip:us`。完整代码列表可参考 [IBAN Country Codes](https://www.iban.com/country-codes)。

Country/region categories use two-letter ISO 3166-1 alpha-2 codes, such as `geoip:cn` and `geoip:us`. See [IBAN Country Codes](https://www.iban.com/country-codes) for the full list.

本项目保留上游额外提供的分类：`geoip:private`、`geoip:cloudflare`、`geoip:cloudfront`、`geoip:facebook`、`geoip:fastly`、`geoip:google`、`geoip:netflix`、`geoip:telegram`、`geoip:twitter`、`geoip:tor`，并新增 `geoip:anycast`。

This project keeps the extra categories provided by upstream: `geoip:private`, `geoip:cloudflare`, `geoip:cloudfront`, `geoip:facebook`, `geoip:fastly`, `geoip:google`, `geoip:netflix`, `geoip:telegram`, `geoip:twitter`, and `geoip:tor`, and additionally adds `geoip:anycast`.

## 与上游的区别 / Differences From Upstream

1. 集成 [bgptools/anycast-prefixes](https://github.com/bgptools/anycast-prefixes) 数据，新增 `geoip:anycast` 标签，并在对应的 `dat`、MMDB、sing-box SRS、mihomo MRS、Clash ruleset 和 Surge ruleset 输出中生成该列表。

   This fork integrates data from [bgptools/anycast-prefixes](https://github.com/bgptools/anycast-prefixes), adds the `geoip:anycast` tag, and generates the list in the corresponding `dat`, MMDB, sing-box SRS, mihomo MRS, Clash ruleset, and Surge ruleset outputs.

2. 基础国家/地区 IP 数据源不再使用 MaxMind GeoLite2，改用 [IPinfo Lite](https://ipinfo.io/developers/ipinfo-lite-database) 免费库；国家/地区和 ASN 相关输出都基于同一个 `ipinfo_lite.mmdb`。

   The base country/region IP dataset uses the free [IPinfo Lite](https://ipinfo.io/developers/ipinfo-lite-database) database instead of MaxMind GeoLite2. Country/region and ASN-related outputs are both generated from the same `ipinfo_lite.mmdb` file.

3. CI 构建下载 IPinfo MMDB，需要在 GitHub Actions secrets 中配置 `IPINFO_TOKEN`。

   CI downloads the IPinfo MMDB during the build, so `IPINFO_TOKEN` must be configured in GitHub Actions secrets.

除以上差异外，项目结构和生成逻辑尽量保持与上游一致，方便后续跟踪上游变化。

Except for the differences above, the project structure and generation flow are kept as close to upstream as possible to make future upstream tracking easier.
