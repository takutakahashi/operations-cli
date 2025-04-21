# Changelog

## [v0.6.12](https://github.com/takutakahashi/operation-mcp/compare/v0.6.11...v0.6.12) - 2025-04-21
- Remove stdout logs for mcp-server by @kommon-ai in https://github.com/takutakahashi/operation-mcp/pull/73

## [v0.6.11](https://github.com/takutakahashi/operation-mcp/compare/v0.6.10...v0.6.11) - 2025-04-20

## [v0.6.10](https://github.com/takutakahashi/operation-mcp/compare/v0.6.9...v0.6.10) - 2025-04-20
- Init manager by @takutakahashi in https://github.com/takutakahashi/operation-mcp/pull/68
- Use mcp go by @takutakahashi in https://github.com/takutakahashi/operation-mcp/pull/69

## [v0.6.9](https://github.com/takutakahashi/operation-mcp/compare/v0.6.8...v0.6.9) - 2025-04-20
- MCP Server Design Document by @devin-ai-integration in https://github.com/takutakahashi/operation-mcp/pull/62
- MCP Server Implementation by @devin-ai-integration in https://github.com/takutakahashi/operation-mcp/pull/64
- Implement MCP server as a cobra subcommand by @devin-ai-integration in https://github.com/takutakahashi/operation-mcp/pull/65

## [v0.6.8](https://github.com/takutakahashi/operation-mcp/compare/v0.6.7...v0.6.8) - 2025-04-20
- issue #57: パラメータ定義をrootツールに移動し、サブツールは参照するよう変更 by @devin-ai-integration in https://github.com/takutakahashi/operation-mcp/pull/58
- Add e2e test for param_refs feature by @devin-ai-integration in https://github.com/takutakahashi/operation-mcp/pull/60
- Fix E2E test by adding required seconds parameter to sleep_short command by @devin-ai-integration in https://github.com/takutakahashi/operation-mcp/pull/61

## [v0.6.7](https://github.com/takutakahashi/operation-mcp/compare/v0.6.6...v0.6.7) - 2025-04-18
- feat: Add S3 remote configuration support by @kommon-ai in https://github.com/takutakahashi/operation-mcp/pull/55

## [v0.6.6](https://github.com/takutakahashi/operation-mcp/compare/v0.6.5...v0.6.6) - 2025-04-18
- Add e2e test for remote config import via HTTP by @takutakahashi in https://github.com/takutakahashi/operation-mcp/pull/52

## [v0.6.5](https://github.com/takutakahashi/operation-mcp/compare/v0.6.4...v0.6.5) - 2025-04-18
- version フラグと upgrade コマンドが config を必要としないよう修正 by @kommon-ai in https://github.com/takutakahashi/operation-mcp/pull/49

## [v0.6.4](https://github.com/takutakahashi/operation-mcp/compare/v0.6.3...v0.6.4) - 2025-04-18
- Fix: インストールスクリプトのアセットパターンとドキュメントの修正 by @kommon-ai in https://github.com/takutakahashi/operation-mcp/pull/46

## [v0.6.3](https://github.com/takutakahashi/operation-mcp/compare/v0.6.2...v0.6.3) - 2025-04-18
- Config に imports 機能を追加 by @kommon-ai in https://github.com/takutakahashi/operation-mcp/pull/43

## [v0.6.2](https://github.com/takutakahashi/operation-mcp/compare/v0.6.1...v0.6.2) - 2025-04-18
- Fix nested subtool recognition in exec command by @kommon-ai in https://github.com/takutakahashi/operation-mcp/pull/38

## [v0.6.1](https://github.com/takutakahashi/operation-mcp/compare/v0.6.0...v0.6.1) - 2025-04-17
- Fix exec flag by @takutakahashi in https://github.com/takutakahashi/operation-mcp/pull/32
- Fix: Change list output format to tool_subtool format (#33) by @kommon-ai in https://github.com/takutakahashi/operation-mcp/pull/35

## [v0.5.2](https://github.com/takutakahashi/operation-mcp/compare/v0.5.1...v0.5.2) - 2025-04-17

## [v0.5.1](https://github.com/takutakahashi/operation-mcp/compare/v0.5.0...v0.5.1) - 2025-04-17

## [v0.5.0](https://github.com/takutakahashi/operation-mcp/compare/v0.4.0...v0.5.0) - 2025-04-17
- use cobra and viper by @takutakahashi in https://github.com/takutakahashi/operation-mcp/pull/29

## [v0.4.0](https://github.com/takutakahashi/operation-mcp/compare/v0.3.0...v0.4.0) - 2025-04-14
- tool_subtool 形式でコマンド実行をサポート (#27) by @kommon-ai in https://github.com/takutakahashi/operation-mcp/pull/28

## [v0.3.0](https://github.com/takutakahashi/operation-mcp/compare/v0.2.0...v0.3.0) - 2025-04-13
- feat: add upgrade command by @kommon-ai in https://github.com/takutakahashi/operation-mcp/pull/24
- インストールスクリプトを作成しGitHub Pagesで公開する by @kommon-ai in https://github.com/takutakahashi/operation-mcp/pull/26

## [v0.2.0](https://github.com/takutakahashi/operation-mcp/compare/v0.1.0...v0.2.0) - 2025-04-13

## [v0.1.0](https://github.com/takutakahashi/operation-mcp/commits/v0.1.0) - 2025-04-12
- Implement Operations CLI Tool by @takutakahashi in https://github.com/takutakahashi/operation-mcp/pull/2
- Implement e2e tests for operation-mcp by @takutakahashi in https://github.com/takutakahashi/operation-mcp/pull/4
- Add CI/CD workflows by @takutakahashi in https://github.com/takutakahashi/operation-mcp/pull/6
- CI/CD ワークフローの修正と改善 by @takutakahashi in https://github.com/takutakahashi/operation-mcp/pull/7
- Add exec command to support flexible tool execution by @kommon-ai in https://github.com/takutakahashi/operation-mcp/pull/9
- [Issue #10] go fmt を通す by @kommon-ai in https://github.com/takutakahashi/operation-mcp/pull/11
- Remote execution via SSH by @kommon-ai in https://github.com/takutakahashi/operation-mcp/pull/16
- SSH リモート実行の E2E テストと go fmt 適用（Issue #17） by @kommon-ai in https://github.com/takutakahashi/operation-mcp/pull/18
- Apply go fmt and fix golint issues by @takutakahashi in https://github.com/takutakahashi/operation-mcp/pull/19
- Fix golint issues: rename ToolInfo to Info by @takutakahashi in https://github.com/takutakahashi/operation-mcp/pull/20
- 設定ファイルにシェルスクリプト埋め込み機能を追加 by @kommon-ai in https://github.com/takutakahashi/operation-mcp/pull/22
