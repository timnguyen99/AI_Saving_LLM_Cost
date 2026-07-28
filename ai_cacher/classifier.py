"""Idea and Co-author: Tim Nguyen (timothynnguyen9@gmail.com)"""

from typing import Optional


class ComplexityClassifier:
    """Heuristic classifier for local vs cloud execution."""

    LOCAL_KEYWORDS = [
        "format", "reformat", "json", "csv", "yaml", "markdown", "summarize",
        "extract", "clean", "normalize", "validate", "transform", "tokenize",
    ]
    CLOUD_KEYWORDS = [
        "javascript", "python", "sql", "code", "complex", "reason", "analysis",
        "multi-step", "strategy", "plan", "problem", "math", "proof", "logic",
    ]

    def is_local_task(self, user_text: str, system_prompt: Optional[str] = None) -> bool:
        combined = " ".join(filter(None, [system_prompt or "", user_text or ""]))
        normalized = combined.lower()

        cloud_score = sum(normalized.count(word) for word in self.CLOUD_KEYWORDS)
        local_score = sum(normalized.count(word) for word in self.LOCAL_KEYWORDS)

        if local_score >= 2 and cloud_score == 0:
            return True
        if cloud_score >= 2 and local_score == 0:
            return False
        if "simple" in normalized or "basic" in normalized:
            return True

        return local_score >= cloud_score
