"""Message、Prompt Template 与流式输出。"""

from langchain.messages import HumanMessage, SystemMessage
from langchain.prompts import ChatPromptTemplate

from model import build_model


def main() -> None:
    model = build_model(streaming=True)
    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", "你是一名严谨的 {language} 教师，回答保持简洁。"),
            ("human", "问题：{question}"),
        ]
    )

    # 模板先生成 Message 列表；它也可以直接与模型组成 Runnable。
    messages = prompt.invoke(
        {"language": "Go", "question": "为什么 context.Context 通常是第一个参数？"}
    ).messages
    for chunk in model.stream(messages):
        # 工具调用或元数据 chunk 可能没有可显示的文本。
        if chunk.text:
            print(chunk.text, end="", flush=True)
    print()

    # Message 可以独立构造，适合需要明确维护角色与历史的代码。
    answer = build_model().invoke(
        [
            SystemMessage("回答必须指出不确定性。"),
            HumanMessage("Go 的 map 可以并发读写吗？"),
        ]
    )
    print(answer.text)


if __name__ == "__main__":
    main()
