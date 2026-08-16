"""一个不依赖模型服务的 LangGraph 审核工作流。

运行后先暂停在 review 节点。传入 True 继续发布；传入 False
会进入 revise 节点并再次暂停，展示受控循环和恢复机制。
"""

from __future__ import annotations

import operator
from typing import Annotated, Literal

from langgraph.checkpoint.memory import InMemorySaver
from langgraph.graph import END, START, StateGraph
from langgraph.types import Command, interrupt
from typing_extensions import TypedDict


class ContentState(TypedDict):
    """图中流动的业务状态。events 用 reducer 追加，而不是覆盖。"""

    topic: str
    plan: str
    draft: str
    status: str
    revision: int
    events: Annotated[list[str], operator.add]


def plan(state: ContentState) -> dict:
    """生成固定计划；替换为模型调用时，输入输出契约无需变化。"""
    return {
        "plan": f"解释 {state['topic']} 的概念、用途和风险边界",
        "events": ["plan：已生成计划"],
    }


def draft(state: ContentState) -> dict:
    """根据计划生成草稿，并将流程推进到待审核状态。"""
    return {
        "draft": f"{state['topic']}：{state['plan']}",
        "status": "pending_review",
        "events": ["draft：已生成草稿，等待审核"],
    }


def review(state: ContentState) -> Command[Literal["publish", "revise"]]:
    """暂停图并把审核资料交给调用方；恢复值决定动态跳转方向。"""
    approved = interrupt(
        {
            "kind": "content_approval",
            "question": "是否批准发布？",
            "draft": state["draft"],
            "revision": state["revision"],
        }
    )
    return Command(
        update={
            "status": "approved" if approved else "rejected",
            "events": [f"review：{'通过' if approved else '退回'}"],
        },
        goto="publish" if approved else "revise",
    )


def revise(state: ContentState) -> dict:
    """被退回后修订，再由静态边回到 review 节点。"""
    revision = state["revision"] + 1
    return {
        "draft": f"{state['draft']}（第 {revision} 次修订：补充可验证示例）",
        "status": "pending_review",
        "revision": revision,
        "events": ["revise：已补充示例，等待再次审核"],
    }


def publish(state: ContentState) -> dict:
    """将真实的发布 API 放在这里；生产代码应使用幂等键保护副作用。"""
    return {"status": "published", "events": ["publish：已发布"]}


def build_graph():
    """声明结构，再在 compile 阶段注入 checkpointer。"""
    builder = StateGraph(ContentState)
    builder.add_node("plan", plan)
    builder.add_node("draft", draft)
    builder.add_node("review", review)
    builder.add_node("revise", revise)
    builder.add_node("publish", publish)

    builder.add_edge(START, "plan")
    builder.add_edge("plan", "draft")
    builder.add_edge("draft", "review")
    builder.add_edge("revise", "review")
    builder.add_edge("publish", END)

    # 内存实现只用于本地学习和测试；服务重启后检查点会丢失。
    return builder.compile(checkpointer=InMemorySaver())


def print_updates(graph, graph_input, config) -> None:
    """以 updates 模式观察每一个节点写入了哪些状态字段。"""
    for update in graph.stream(graph_input, config=config, stream_mode="updates"):
        print(update)


if __name__ == "__main__":
    graph = build_graph()
    config = {"configurable": {"thread_id": "content-001"}}
    initial_state: ContentState = {
        "topic": "LangGraph",
        "plan": "",
        "draft": "",
        "status": "new",
        "revision": 0,
        "events": [],
    }

    print("第一次执行：工作流将在 review 节点中断。")
    print_updates(graph, initial_state, config)

    print("\n恢复执行：批准发布。")
    print_updates(graph, Command(resume=True), config)

    final_state = graph.get_state(config).values
    print("\n最终状态：", final_state["status"])
    print("事件记录：")
    for event in final_state["events"]:
        print("-", event)
