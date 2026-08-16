"""create_agent 处理模型选择工具、执行工具和继续回答的循环。"""

from langchain.agents import create_agent
from langchain.tools import tool

from model import build_model


@tool
def get_office_hours(day: str) -> str:
    """查询服务台在指定星期的开放时间，day 使用星期一至星期日。"""
    hours = {"星期一": "09:00-18:00", "星期六": "10:00-16:00"}
    return hours.get(day, "当天不开放")


def main() -> None:
    agent = create_agent(
        model=build_model(),
        tools=[get_office_hours],
        system_prompt="你是服务台助手。涉及开放时间时优先调用工具，不要编造。",
    )
    result = agent.invoke(
        {"messages": [{"role": "user", "content": "星期六几点可以办理业务？"}]}
    )
    print(result["messages"][-1].text)


if __name__ == "__main__":
    main()
