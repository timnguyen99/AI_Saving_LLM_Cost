from typing import Any, Dict, List

from ai_cacher import AICacherWrapper, SemanticCacheRouter


class MockOpenAIChat:
    def __init__(self):
        self.call_count = 0

    def create(self, messages: List[Dict[str, str]], **kwargs: Any) -> Dict[str, Any]:
        self.call_count += 1
        user_prompt = "".join([m.get("content", "") for m in messages if m.get("role") == "user"])
        response_text = self._generate_response(user_prompt)
        return {
            "choices": [
                {
                    "message": {
                        "role": "assistant",
                        "content": response_text,
                    }
                }
            ],
            "usage": {
                "prompt_tokens": self._count_tokens(user_prompt),
                "completion_tokens": self._count_tokens(response_text),
            },
        }

    @staticmethod
    def _generate_response(prompt: str) -> str:
        return f"AI response for prompt: {prompt[:120]}"

    @staticmethod
    def _count_tokens(text: str) -> int:
        return max(1, len(text.split()))


class MockOpenAIClient:
    def __init__(self):
        self.chat = type("obj", (), {"completions": MockOpenAIChat()})


def count_tokens(payload: Dict[str, Any]) -> int:
    usage = payload.get("usage", {})
    return int(usage.get("prompt_tokens", 0)) + int(usage.get("completion_tokens", 0))


def run_direct(client: MockOpenAIClient, messages: List[Dict[str, str]]) -> Dict[str, Any]:
    response = client.chat.completions.create(messages=messages)
    response["cached_by"] = "Direct-Cloud"
    response["latency_ms"] = 1500
    return response


def run_cached(wrapper: AICacherWrapper, messages: List[Dict[str, str]]) -> Dict[str, Any]:
    return wrapper.create_completion(messages=messages)


def main() -> None:
    messages_batch = [
        [
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": "Summarize the monthly sales report."},
        ],
        [
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": "Summarize the monthly sales report."},
        ],
        [
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": "Convert this list to JSON: apple, banana, cherry."},
        ],
        [
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": "Summarize the monthly sales report."},
        ],
        [
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": "Convert this list to JSON: apple, banana, cherry."},
        ],
    ]

    mock_client = MockOpenAIClient()
    router = SemanticCacheRouter(threshold=0.88)
    cached_wrapper = AICacherWrapper(mock_client, router)

    direct_tokens = 0
    cached_cloud_tokens = 0
    cached_hits = 0
    direct_calls = 0
    cached_calls = 0

    print("=== Direct cloud usage ===")
    for messages in messages_batch:
        result = run_direct(mock_client, messages)
        direct_calls += 1
        tokens = count_tokens(result)
        direct_tokens += tokens
        print(f"Call {direct_calls}: prompt='{messages[-1]['content']}' tokens={tokens}")

    print("\n=== Cached usage ===")
    for messages in messages_batch:
        result = run_cached(cached_wrapper, messages)
        if result.get("cached_by") == "AI-Cacher-Local":
            cached_hits += 1
            tokens = 0
        else:
            cached_calls += 1
            tokens = count_tokens(result)
            cached_cloud_tokens += tokens
        print(
            f"Request: '{messages[-1]['content']}' cached_by={result.get('cached_by')} tokens={tokens}")

    print("\n=== Summary ===")
    print(f"Direct cloud calls: {direct_calls}")
    print(f"Cached cloud calls: {cached_calls}")
    print(f"Local cache hits: {cached_hits}")
    print(f"Total direct tokens: {direct_tokens}")
    print(f"Total cached cloud tokens: {cached_cloud_tokens}")
    print(f"Token reduction: {direct_tokens - cached_cloud_tokens} tokens")
    print(
        f"Estimated cost savings: ${(direct_tokens - cached_cloud_tokens) * 0.000002:.6f} "
        "(assuming $0.002 per 1,000 tokens)"
    )


if __name__ == "__main__":
    main()
