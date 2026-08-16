"""以 Pydantic schema 接收模型结构化输出。"""

from pydantic import BaseModel, Field

from model import build_model


class Ticket(BaseModel):
    """Field 描述会成为模型可见的 schema 一部分。"""

    title: str = Field(description="不超过 20 字的标题")
    priority: str = Field(description="只能是 low、medium 或 high")
    tags: list[str] = Field(description="用于检索的短标签")


def main() -> None:
    extractor = build_model().with_structured_output(Ticket)
    ticket = extractor.invoke("登录后页面空白，所有用户均受影响，需要尽快处理。")
    print(ticket.model_dump_json(indent=2))


if __name__ == "__main__":
    main()
