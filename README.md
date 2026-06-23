# eino-claude-code

Claude Code CLI 作为 [eino](https://github.com/cloudwego/eino) 框架的一等公民 Agent。

把 Claude Code 变成一个标准的 eino `Agent` 和 `Tool`，可以直接用 `Runner.Run()`、参与多 Agent 协作、被 callback/middleware 观测，**就像用 eino 自带的 ChatModelAgent 一样**。

```go
// 和任何 eino agent 完全相同的用法
agent, _ := claudecode.New(claudecode.WithMaxTurns(5))
runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
events := runner.Run(ctx, []adk.Message{schema.UserMessage("Hello!")})
```

## 安装

```bash
go get github.com/239what475/eino-claude-code
```

**前置条件**：`claude` CLI 已安装并在 `$PATH` 中。

## 核心概念

Claude Code CLI 是一个**自包含的完整 Agent**——它内部有自己的 ReAct 循环、工具执行、会话管理。eino-claude-code 不做的事情：把 CLI 拆开、模拟 ChatModel。它做的事情：给 CLI 套上 eino 的 Agent 接口，使其能无缝融入 eino 的多 Agent 编排、流式输出、中断恢复、可观测性体系。

## 五种集成模式

### 模式 1：直接 Agent（最简单）

```go
agent, _ := claudecode.New(
    claudecode.WithSystemPrompt("You are a Go expert."),
    claudecode.WithTools("Read", "Write", "Bash"),
    claudecode.WithModel("claude-sonnet-4-6"),
    claudecode.WithMaxTurns(10),
    claudecode.WithPermissionMode("acceptEdits"),
)

runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
events := runner.Run(ctx, []adk.Message{
    schema.UserMessage("Refactor the authentication module."),
})
```

**适用场景**：把 Claude Code 当作一个独立 Agent，直接对话。

**流式输出**：

```go
runner := adk.NewRunner(ctx, adk.RunnerConfig{
    Agent:          agent,
    EnableStreaming: true, // 开启流式
})
events := runner.Run(ctx, messages)
for {
    evt, ok := events.Next()
    if !ok { break }
    if evt.Output.MessageOutput.IsStreaming {
        for {
            chunk, err := evt.Output.MessageOutput.MessageStream.Recv()
            if err == io.EOF { break }
            fmt.Print(chunk.Content) // 逐块输出
        }
    }
}
```

### 模式 2：作为 Tool 被 eino Agent 调度（推荐）

eino 的 ChatModelAgent 做"大脑"（决策调用哪个工具），Claude Code 做"双手"（执行复杂任务）。

```go
// 把 Claude Code 包装成一个 Tool
ccTool, _ := claudecode.NewTool(
    claudecode.WithName("claude_code"),
    claudecode.WithDescription(
        "Delegate complex multi-step tasks to Claude Code. "+
        "Use for: file operations, shell commands, code analysis, git, web search.",
    ),
    claudecode.WithMaxTurns(10),
    claudecode.WithAllowedTools("Read", "Write", "Edit", "Bash", "Glob", "Grep"),
)

// 创建一个 ChatModelAgent（用任何 eino ChatModel：OpenAI、Claude API 等）
supervisor, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "supervisor",
    Description: "Orchestrates complex tasks by delegating to Claude Code.",
    Instruction: "You are a supervisor. Delegate complex tasks to claude_code.",
    Model:       myChatModel, // 你的 ChatModel
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{ccTool},
        },
    },
    MaxIterations: 5,
})

runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: supervisor})
events := runner.Run(ctx, []adk.Message{
    schema.UserMessage("Analyze the performance of the auth module and suggest improvements."),
})
```

**执行流程**：User → ChatModelAgent 决策 → 调用 `claude_code` tool → Claude Code CLI 执行（读文件、跑命令、分析） → 结果返回给 ChatModelAgent → 继续对话或结束。

**适用场景**：需要"思考+执行"分离。ChatModel 做决策，Claude Code 做具体工作。

### 模式 3：多 Agent 协作

ClaudeCodeAgent 可以作为子 Agent 参与 eino 的 `DeepAgent` 和 `PlanExecute` 模式。

```go
// DeepAgent：ChatModel 规划，ClaudeCodeAgent 执行
executor, _ := claudecode.New(
    claudecode.WithName("executor"),
    claudecode.WithDescription("Executes tasks using Claude Code CLI."),
    claudecode.WithMaxTurns(10),
)

deepAgent, _ := deep.New(ctx, &deep.Config{
    Name:        "deep-demo",
    Description: "Deep reasoning agent with Claude Code executor.",
    ChatModel:   myChatModel,
    SubAgents:   []adk.Agent{executor}, // ClaudeCodeAgent 作为子 Agent
    MaxIteration: 5,
})

// PlanExecute：ClaudeCodeAgent 可以担任 Planner、Executor、Replanner 任一角色
pe, _ := planexecute.New(ctx, &planexecute.Config{
    Planner:   plannerAgent,
    Executor:  executor,      // ClaudeCodeAgent 执行计划
    Replanner: replannerAgent,
})
```

**适用场景**：复杂的多步骤任务，需要规划-执行-反思循环。

### 模式 4：AgentTool 包装

用 `adk.NewAgentTool` 把 ClaudeCodeAgent 包装成标准 Tool，供任何 eino Agent 调用。

```go
agent, _ := claudecode.New(...)
agentTool := adk.NewAgentTool(ctx, agent)
// agentTool 现在是标准的 tool.BaseTool，可以加入任何 ToolsConfig
```


## 配置选项

### 基础配置

| Option | 说明 |
|--------|------|
| `WithName(name)` | Agent 名称（默认 "claude-code"） |
| `WithDescription(desc)` | Agent 描述 |
| `WithBin(path)` | CLI 路径（默认自动搜索） |

### Prompt

| Option | 说明 |
|--------|------|
| `WithSystemPrompt(prompt)` | 系统提示词（替换默认） |
| `WithAppendSystemPrompt(prompt)` | 追加到默认系统提示词 |

### 工具

| Option | 说明 |
|--------|------|
| `WithTools(names...)` | 可用工具列表（nil=默认，空=无工具） |
| `WithAllowedTools(names...)` | 自动批准的工具 |
| `WithDisallowedTools(names...)` | 禁用的工具 |

### 模型

| Option | 说明 |
|--------|------|
| `WithModel(model)` | 模型名称，如 `"claude-sonnet-4-6"` |
| `WithFallbackModel(model)` | 备用模型 |
| `WithEffort(level)` | 思考力度：`"low"`, `"medium"`, `"high"` |

### 预算

| Option | 说明 |
|--------|------|
| `WithMaxTurns(n)` | 最大回合数（0=默认） |
| `WithMaxBudgetUSD(n)` | 费用上限（美元） |

### 权限

| Option | 说明 |
|--------|------|
| `WithPermissionMode(mode)` | `"default"`, `"acceptEdits"`, `"plan"`, `"bypassPermissions"` |

### 会话

| Option | 说明 |
|--------|------|
| `WithContinueConversation()` | 恢复最近的对话 |
| `WithResume(sessionID)` | 恢复指定 session |
| `WithSessionID(id)` | 创建命名 session |
| `WithForkSession()` | 复制 session 到新 ID |

### MCP

| Option | 说明 |
|--------|------|
| `WithMCPConfig(json)` | MCP 配置（内联 JSON） |
| `WithMCPConfigPath(path)` | MCP 配置文件路径 |

### 高级

| Option | 说明 |
|--------|------|
| `WithCWD(dir)` | CLI 进程工作目录 |
| `WithEnv(kv...)` | 环境变量（`KEY=VALUE`） |
| `WithStderr(fn)` | Stderr 回调（每行） |
| `WithEmitToolEvents()` | 暴露 Claude Code 内部的工具调用为 AgentEvent |
| `WithStructuredOutput(schema)` | JSON Schema 约束输出 |
| `WithBetas(betas...)` | 启用 beta 功能 |
| `WithSettings(json)` | 内联设置 JSON |
| `WithAddDirs(dirs...)` | 添加工作目录 |
| `WithIncludePartialMessages()` | 请求部分消息流事件 |
| `WithExtraArgs(args)` | 透传额外 CLI 参数 |

### eino 集成

| Option | 说明 |
|--------|------|

## 与 eino 生态的兼容性

### 完全兼容

| eino 功能 | 使用方式 |
|-----------|---------|
| `Runner.Run()` / `Runner.Query()` | 和 ChatModelAgent 完全一样 |
| `callbacks.Handler` | `OnStart/OnEnd/OnError` 回调，通过 `Runner.Query(ctx, prompt, WithCallbacks(h))` |
| `Tool Middleware` | `ToolCallMiddlewares` 可包装 ClaudeCodeTool 的每次调用 |
| `AgentTool` | `adk.NewAgentTool(ctx, ccAgent)` → 标准 `tool.BaseTool` |
| `ResumableAgent` | 实现 `adk.ResumableAgent`，支持 eino checkpoint/interrupt |
| `DeepAgent` | `SubAgents: []adk.Agent{ccAgent}` |
| `PlanExecute` | Planner / Executor / Replanner 均可用 ClaudeCodeAgent |
| Streaming | `EnableStreaming: true` → `MessageStream` 流式输出 |
| Session 管理 | `WithResume` / `WithContinue` / `WithSessionID` |

### 间接可用（通过 Tool 模式）

这些 eino 功能设计给内部有 ChatModel 的 Agent，但当 ClaudeCodeAgent 作为 Tool 被 ChatModelAgent 调用时，ChatModelAgent 的这些功能仍然全部生效：

| eino 功能 | 说明 |
|-----------|------|
| `ChatModelAgentMiddleware` | ChatModelAgent 的 middleware 可以观测到 claude_code 工具调用 |
| `Graph / Chain` | 通过 AgentTool → ChatModelAgent → compose.Graph 间接参与编排 |
| `Summarization` middleware | 上下文超限时自动摘要（ChatModelAgent 层面） |
| `Skill` middleware | 技能模板加载（ChatModelAgent 层面） |

### 有意的架构差异（不是缺陷）

| 方面 | ClaudeCodeAgent | ChatModelAgent | 原因 |
|------|----------------|----------------|------|
| 工具执行 | CLI 内部执行 | eino `ToolsNode` 执行 | Claude Code 是完整 Agent，自带工具生态 |
| 模型调用 | CLI 管理 | eino `ChatModel` 接口 | Claude Code 自己选模型、管 retry |
| ReAct 循环 | CLI 内部控制 | eino `compose.Graph` 控制 | 两种有效架构，互不干扰 |
| Prompt 模板 | CLI 的 system prompt | eino `ChatTemplate` | CLI 有自己的 prompt 体系 |

## 局限性

### 架构层面

- **不能作为 ChatModel 使用**。Claude Code 是 Agent，不是 Token 生成器。不要把它塞进 `model.BaseChatModel` 接口——那是错误的抽象。
- **不能替换 ChatModelAgent 内部的 ChatModel**。Claude Code 自己管理模型调用、重试、工具执行。eino 的 `ChatModelAgentMiddleware.WrapModel` 对它无效。
- **多 Agent transfer 不完全**。Supervisor prebuilt 依赖 eino 内部的 transfer 机制，和 CLI 的 Task tool 不完全兼容。推荐用 AgentTool 模式替代。

### 运行环境



### 与 ChatModelAgent 的行为差异

- **不经过 eino ChatModel 调用链**。Callbacks 在 Agent 层面触发（OnStart:Agent/claude-code），不在 ChatModel 层面。
- **会话状态在 CLI 内部**。eino 的 session event history 和 CLI 的 `--session-id` 是两条独立的线。用 `WithResume` 来桥接。
- **Interrupt/Resume 依赖 CLI session**。中断后恢复需要 CLI session ID，我们已经在 `Resume()` 中处理了状态序列化。

## 项目结构

```
├── agent.go              # ClaudeCodeAgent (adk.Agent + adk.ResumableAgent)
├── agent_test.go         # 单元测试（mock CLI）
├── args.go               # FindCLI + BuildArgs
├── cli.go                # CLI 子进程管理（Simple + Streaming 模式）
├── convertor.go          # CLI JSON ↔ eino AgentEvent 转换
├── errors.go             # CLIError, AgentError, sentinel errors
├── interrupt.go          # ResumableAgent 实现（checkpoint 状态序列化）
├── options.go            # Options 结构
├── options_funcs.go      # 35 个 With* 配置函数
├── session.go            # NewSessionID
├── tool.go               # ClaudeCodeTool (tool.InvokableTool)
├── types.go              # CLI JSON 消息类型
├── examples/
│   ├── helloworld/       # 最简示例（流式输出 + session）
│   ├── tool/             # ChatModelAgent 调度 ClaudeCodeTool
└── README.md
```

## 与相关项目的对比

| | eino-claude-code | claude-agent-sdk-go | trpc-agent-go ClaudeCodeAgent |
|---|---|---|---|
| 定位 | eino Agent 适配器 | 通用 Go SDK | tRPC Agent 适配器 |
| 集成方式 | 实现 `adk.Agent` | 独立 SDK | 实现 `agent.Agent` |
| 流式输出 | ✅ MessageStream | ✅ iter.Seq2 | ❌ 批量 |
| eino 多 Agent | ✅ | ❌ | ❌ |
| eino Callback | ✅ | ❌ | ❌ |
| MCP 管理 | ⚠️ 配置透传 | ✅ 完整 API | ⚠️ 透传 |
| 依赖 | eino + uuid | 零依赖 | tRPC 框架 |
| CLI 路径发现 | ✅ | ✅ | ❌ |
| Stderr 回调 | ✅ | ✅ | ❌ |

## License

MIT
