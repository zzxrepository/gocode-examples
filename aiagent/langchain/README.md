# LangChain Python 最小示例

示例对应 LangChain 教程中的 Message、Prompt、流式输出、结构化输出和 Agent Tool Calling。默认使用 `ChatOpenAI`；只要目标服务兼容 OpenAI Chat Completions，也可以通过 `OPENAI_BASE_URL` 切换。

## 准备环境

```bash
cd gocode-examples/aiagent/langchain
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
cp .env.example .env
```

shell 不会自动读取 `.env`；可按实际环境导出变量，或使用自己的环境变量管理方式：

```bash
export OPENAI_API_KEY='replace-with-your-key'
export OPENAI_MODEL='gpt-4.1-mini'
# export OPENAI_BASE_URL='https://your-openai-compatible-endpoint/v1'
```

## 运行

```bash
python chat.py
python structured_output.py
python agent_tools.py
```

`agent_tools.py` 中的开放时间数据是本地固定数据，不会访问网络。示例用于观察模型请求工具、运行时执行工具、工具结果回填模型的完整闭环。真实带副作用工具仍必须在工具实现内部做鉴权、参数校验、超时、审计和确认。
