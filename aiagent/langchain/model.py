"""所有示例共用的模型创建函数。"""

import os

from langchain_openai import ChatOpenAI


def build_model(*, streaming: bool = False) -> ChatOpenAI:
    """从环境变量创建 OpenAI 或 OpenAI 兼容聊天模型。"""
    return ChatOpenAI(
        model=os.getenv("OPENAI_MODEL", "gpt-4.1-mini"),
        temperature=0,
        streaming=streaming,
        # None 使用 Provider 默认地址，兼容服务才需要显式 URL。
        base_url=os.getenv("OPENAI_BASE_URL") or None,
    )
