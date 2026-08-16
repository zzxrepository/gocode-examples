# LangGraph 审核工作流

这个示例只使用普通 Python 函数，不需要模型 API Key，用来观察 LangGraph 的核心运行机制：

    START → plan → draft → review ──批准──→ publish → END
                              │
                              └─退回→ revise ───────────────┘

- ContentState 定义节点共享状态；
- events 使用 Annotated[..., operator.add] reducer 追加记录；
- review 通过 interrupt() 暂停，Command(resume=...) 恢复；
- InMemorySaver 按 thread_id 保存检查点；
- stream(..., stream_mode="updates") 逐节点输出状态更新。

## 运行

    python -m venv .venv
    source .venv/bin/activate
    pip install -e .
    python approval_workflow.py

示例中的内存 checkpointer 会在进程结束后清空。真实服务需要使用持久化 checkpointer，并设计状态保留策略、鉴权、幂等写入和人工审批界面。
