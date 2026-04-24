# GeoIP

本项目是 [Loyalsoldier/geoip](https://github.com/Loyalsoldier/geoip) 的分叉版本。基础功能、输出格式、使用方式和上游更新说明请直接参考上游项目。

This project is a fork of [Loyalsoldier/geoip](https://github.com/Loyalsoldier/geoip). Please refer to the upstream project for the base features, output formats, usage, and release notes.

## 与上游的区别 / Differences From Upstream

1. 集成 [bgptools/anycast-prefixes](https://github.com/bgptools/anycast-prefixes) 数据，新增 `geoip:anycast` 标签，并在对应的 `dat`、MMDB、sing-box SRS、mihomo MRS、Clash ruleset 和 Surge ruleset 输出中生成该列表。

   This fork integrates data from [bgptools/anycast-prefixes](https://github.com/bgptools/anycast-prefixes), adds the `geoip:anycast` tag, and generates the list in the corresponding `dat`, MMDB, sing-box SRS, mihomo MRS, Clash ruleset, and Surge ruleset outputs.

2. 基础国家/地区 IP 数据源不再使用 MaxMind GeoLite2，改用 [IPinfo Lite](https://ipinfo.io/developers/ipinfo-lite-database) 免费库；国家/地区和 ASN 相关输出都基于同一个 `ipinfo_lite.mmdb`。

   The base country/region IP dataset uses the free [IPinfo Lite](https://ipinfo.io/developers/ipinfo-lite-database) database instead of MaxMind GeoLite2. Country/region and ASN-related outputs are both generated from the same `ipinfo_lite.mmdb` file.

3. CI 构建下载 IPinfo MMDB，需要在 GitHub Actions secrets 中配置 `IPINFO_TOKEN`。

   CI downloads the IPinfo MMDB during the build, so `IPINFO_TOKEN` must be configured in GitHub Actions secrets.

除以上差异外，项目结构和生成逻辑尽量保持与上游一致，方便后续跟踪上游变化。

Except for the differences above, the project structure and generation flow are kept as close to upstream as possible to make future upstream tracking easier.
