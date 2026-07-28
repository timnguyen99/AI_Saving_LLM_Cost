"""Idea and Co-author: Tim Nguyen (timothynnguyen9@gmail.com)"""

import time
from typing import Any, Dict, List, Optional

from .classifier import ComplexityClassifier
from .router import SemanticCacheRouter


class AICacherWrapper:
    """Wraps an OpenAI-like client to intercept chat completion requests."""

    def __init__(
        self,
        real_openai_client: Any,
        router: SemanticCacheRouter,
        simple_model_name: str = "gpt-mini",
        complex_model_name: str = "gpt-4",
        classifier: Optional[ComplexityClassifier] = None,
    ):
        self.client = real_openai_client
        self.router = router
        self.simple_model_name = simple_model_name
        self.complex_model_name = complex_model_name
        self.classifier = classifier or ComplexityClassifier()

    def _extract_user_prompt(self, messages: List[Dict[str, str]]) -> str:
        return "".join([m.get("content", "") for m in messages if m.get("role") == "user"])

    def _serialize_response(self, cloud_response: Any) -> Dict[str, Any]:
        if hasattr(cloud_response, "model_dump"):
            return cloud_response.model_dump()
        if hasattr(cloud_response, "to_dict"):
            return cloud_response.to_dict()
        if isinstance(cloud_response, dict):
            return cloud_response
        return {"raw": str(cloud_response)}

    def _choose_model(self, user_prompt: str, kwargs: Dict[str, Any]) -> Optional[str]:
        if "model" in kwargs or "model_name" in kwargs:
            return None
        if self.classifier.is_local_task(user_prompt):
            return self.simple_model_name
        return self.complex_model_name

    def create_completion(self, messages: List[Dict[str, str]], **kwargs: Any) -> Dict[str, Any]:
        user_prompt = self._extract_user_prompt(messages)

        cached_response = self.router.check_cache(user_prompt)
        if cached_response is not None:
            cached_response = dict(cached_response)
            cached_response["cached_by"] = "AI-Cacher-Local"
            cached_response["latency_ms"] = 5.0
            return cached_response

        selected_model = self._choose_model(user_prompt, kwargs)
        if selected_model is not None:
            kwargs.setdefault("model", selected_model)

        start_time = time.time()

        if hasattr(self.client, "chat") and hasattr(self.client.chat, "completions"):
            cloud_response = self.client.chat.completions.create(messages=messages, **kwargs)
        elif hasattr(self.client, "ChatCompletion"):
            cloud_response = self.client.ChatCompletion.create(messages=messages, **kwargs)
        else:
            raise RuntimeError("Unsupported OpenAI client interface")

        duration = (time.time() - start_time) * 1000.0
        response_dict = self._serialize_response(cloud_response)
        response_dict["cached_by"] = "Cloud-Server"
        response_dict["latency_ms"] = duration
        response_dict["selected_model"] = selected_model or kwargs.get("model")

        self.router.update_cache(user_prompt, response_dict)
        return response_dict

    def create(self, *args: Any, **kwargs: Any) -> Dict[str, Any]:
        return self.create_completion(*args, **kwargs)
