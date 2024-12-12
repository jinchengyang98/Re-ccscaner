# 贡献指南

感谢你对 CCScanner 项目的关注！我们欢迎各种形式的贡献,包括但不限于:

- 报告问题
- 提交功能请求
- 提交代码修改
- 改进文档
- 分享使用经验

## 目录

- [开发环境设置](#开发环境设置)
- [代码风格](#代码风格)
- [提交代码](#提交代码)
- [测试](#测试)
- [文档](#文档)
- [发布流程](#发布流程)

## 开发环境设置

1. 安装依赖
   ```bash
   # 安装 Go 1.21 或更高版本
   go version

   # 克隆仓库
   git clone https://github.com/yourusername/ccscanner.git
   cd ccscanner

   # 安装依赖
   go mod download
   ```

2. 安装开发工具
   ```bash
   # 安装 golangci-lint
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

   # 安装 goimports
   go install golang.org/x/tools/cmd/goimports@latest
   ```

3. 配置 IDE
   - 推荐使用 VS Code 或 GoLand
   - 安装 Go 插件
   - 配置代码格式化工具

## 代码风格

我们使用标准的 Go 代码风格,并有一些额外的要求:

1. 代码格式化
   ```bash
   # 格式化代码
   goimports -w .
   
   # 运行 linter
   golangci-lint run
   ```

2. 命名规范
   - 使用有意义的变量名
   - 遵循 Go 的命名惯例
   - 避免缩写(除非是常用缩写)

3. 注释规范
   - 所有导出的类型和函数必须有文档注释
   - 复杂的逻辑需要添加注释说明
   - 使用中文注释

4. 错误处理
   - 使用有意义的错误信息
   - 使用错误包装添加上下文
   - 避免忽略错误

5. 代码组织
   - 相关的代码放在同一个包中
   - 包的大小要适中
   - 避免循环依赖

## 提交代码

1. 创建分支
   ```bash
   # 更新主分支
   git checkout main
   git pull

   # 创建特性分支
   git checkout -b feature/your-feature
   ```

2. 提交规范
   - 使用有意义的提交信息
   - 一个提交只做一件事
   - 提交信息格式:
     ```
     类型: 简短的描述

     详细的说明(可选)
     ```
   - 类型包括:
     - feat: 新功能
     - fix: 修复bug
     - docs: 文档更新
     - style: 代码格式
     - refactor: 重构
     - test: 测试相关
     - chore: 构建/工具相关

3. 提交前检查
   ```bash
   # 运行测试
   go test ./...

   # 运行 linter
   golangci-lint run

   # 检查代码格式
   goimports -d .
   ```

4. 创建 Pull Request
   - 填写完整的 PR 描述
   - 关联相关的 Issue
   - 等待 CI 检查通过
   - 请求代码审查

## 测试

1. 单元测试
   ```bash
   # 运行所有测试
   go test ./...

   # 运行特定包的测试
   go test ./pkg/extractor

   # 生成测试覆盖率报告
   go test -coverprofile=coverage.out ./...
   go tool cover -html=coverage.out
   ```

2. 基准测试
   ```bash
   # 运行基准测试
   go test -bench=. ./...

   # 生成性能分析
   go test -bench=. -cpuprofile=cpu.prof ./...
   go tool pprof cpu.prof
   ```

3. 集成测试
   ```bash
   # 运行集成测试
   go test -tags=integration ./...
   ```

4. 测试规范
   - 每个包都应该有测试
   - 测试覆盖率应该达到 80% 以上
   - 包含正常和错误情况的测试
   - 使用表驱动测试
   - 测试应该是可重复的

## 文档

1. 代码文档
   - 所有导出的类型和函数必须有文档注释
   - 使用 godoc 格式的注释
   - 包含使用示例

2. API 文档
   - 更新 API 文档
   - 添加新功能的示例
   - 说明配置选项

3. 用户文档
   - 更新 README.md
   - 添加使用教程
   - 更新常见问题

4. 文档检查
   ```bash
   # 检查文档格式
   markdownlint docs/

   # 预览文档
   godoc -http=:6060
   ```

## 发布流程

1. 版本号规范
   - 遵循语义化版本
   - 格式: vX.Y.Z
   - X: 主版本号(不兼容的更改)
   - Y: 次版本号(向后兼容的功能)
   - Z: 修订号(向后兼容的修复)

2. 发布步骤
   ```bash
   # 更新版本号
   VERSION=v1.0.0

   # 创建标签
   git tag -a $VERSION -m "Release $VERSION"

   # 推送标签
   git push origin $VERSION
   ```

3. 发布检查清单
   - [ ] 所有测试通过
   - [ ] 文档已更新
   - [ ] CHANGELOG.md 已更新
   - [ ] 版本号已更新
   - [ ] 依赖已更新
   - [ ] CI/CD 通过

## 问题反馈

1. 提交 Issue
   - 使用 Issue 模板
   - 提供详细的复现步骤
   - 包含环境信息
   - 附加相关日志

2. 问题分类
   - bug: 程序错误
   - feature: 功能请求
   - docs: 文档相关
   - question: 使用问题

## 社区交流

- GitHub Discussions
- Slack 频道
- 邮件列表
- 微信群

## 行为准则

我们采用 [Contributor Covenant](https://www.contributor-covenant.org/) 作为行为准则。

## 许可证

本项目采用 MIT 许可证。提交代码即表示你同意将代码以 MIT 许可证发布。

## 致谢

感谢所有贡献者的付出！

## 更多资源

- [Go 编程语言规范](https://golang.org/ref/spec)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go 测试指南](https://golang.org/doc/tutorial/add-a-test) 