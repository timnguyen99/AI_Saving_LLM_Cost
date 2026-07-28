"""Idea and Co-author: Tim Nguyen (timothynnguyen9@gmail.com)"""

import hashlib
import logging
import math
from datetime import datetime
from typing import Any, Dict, List, Optional, Tuple

try:
    import numpy as np
except ImportError:  # pragma: no cover
    np = None

try:
    from sentence_transformers import SentenceTransformer
except ImportError:  # pragma: no cover
    SentenceTransformer = None

try:
    import pymongo
    from pymongo.collection import Collection
except ImportError:  # pragma: no cover
    pymongo = None
    Collection = None

logger = logging.getLogger(__name__)


class SemanticCacheRouter:
    """Semantic cache router for local embedding matching and optional MongoDB persistence."""

    def __init__(
        self,
        threshold: float = 0.95,
        model_name: str = "all-MiniLM-L6-v2",
        mongo_uri: Optional[str] = None,
        mongo_db_name: str = "ai_cacher",
        mongo_collection_name: str = "prompt_cache",
    ):
        self.threshold = threshold
        self.cache: Dict[int, Tuple[np.ndarray, str, Any]] = {}
        self.counter = 0
        self.model_name = model_name
        self._model = None
        self._mongo_collection: Optional[Collection] = None

        if mongo_uri is not None:
            if pymongo is None:
                raise RuntimeError(
                    "pymongo is required for MongoDB cache support. "
                    "Install with `pip install pymongo`."
                )
            client = pymongo.MongoClient(mongo_uri)
            db = client[mongo_db_name]
            self._mongo_collection = db[mongo_collection_name]
            self._mongo_collection.create_index("prompt_hash")

    @staticmethod
    def _prompt_hash(prompt: str) -> str:
        return hashlib.sha256((prompt or "").encode("utf-8")).hexdigest()

    def _load_model(self):
        if self._model is not None:
            return self._model
        if SentenceTransformer is None:
            raise RuntimeError(
                "sentence-transformers is required for local embedding generation. "
                "Install with `pip install sentence-transformers` or use the fallback model."
            )
        self._model = SentenceTransformer(self.model_name)
        return self._model

    def _get_embedding(self, text: str) -> np.ndarray:
        text = text or ""
        if SentenceTransformer is not None:
            model = self._load_model()
            embedding = model.encode([text], normalize_embeddings=True)[0]
            if np is not None:
                return np.asarray(embedding, dtype=np.float32)
            return embedding.tolist()

        tokens = text.lower().split()
        freq: Dict[str, int] = {}
        for token in tokens:
            freq[token] = freq.get(token, 0) + 1
        if not freq:
            return [0.0] * 512 if np is None else np.zeros(512, dtype=np.float32)

        if np is None:
            vector = [0.0] * 512
            for i, token in enumerate(sorted(freq.keys())):
                vector[i % 512] = float(freq[token])
            norm = math.sqrt(sum(x * x for x in vector))
            if norm > 0:
                return [x / norm for x in vector]
            return vector

        vector = np.zeros(512, dtype=np.float32)
        for i, token in enumerate(sorted(freq.keys())):
            vector[i % 512] = freq[token]
        norm = np.linalg.norm(vector)
        return vector / norm if norm > 0 else vector

    def clear_cache(self) -> None:
        self.cache.clear()
        self.counter = 0
        if self._mongo_collection is not None:
            self._mongo_collection.delete_many({})

    def _load_db_documents(self) -> List[Dict[str, Any]]:
        if self._mongo_collection is None:
            return []
        return list(self._mongo_collection.find({}))

    def _find_best_match(self, new_vec: Any, docs: List[Dict[str, Any]]) -> Tuple[float, Optional[Dict[str, Any]]]:
        best_score = 0.0
        best_payload = None

        for doc in docs:
            cached_vec = doc.get("embedding", [])
            if np is not None:
                cached_vec = np.asarray(cached_vec, dtype=np.float32)
                denom = np.linalg.norm(new_vec) * np.linalg.norm(cached_vec)
                similarity = np.dot(new_vec, cached_vec) / denom if denom > 0 else 0.0
            else:
                denom = math.sqrt(sum(x * x for x in new_vec)) * math.sqrt(sum(x * x for x in cached_vec))
                similarity = sum(x * y for x, y in zip(new_vec, cached_vec)) / denom if denom > 0 else 0.0
            if similarity > best_score:
                best_score = similarity
                best_payload = doc.get("response")

        return best_score, best_payload

    def check_cache(self, prompt: str) -> Optional[Dict[str, Any]]:
        if self._mongo_collection is not None:
            prompt_hash = self._prompt_hash(prompt)
            exact_doc = self._mongo_collection.find_one({"prompt_hash": prompt_hash})
            if exact_doc is not None:
                self._mongo_collection.update_one(
                    {"_id": exact_doc["_id"]},
                    {"$set": {"last_used": datetime.utcnow()}},
                )
                return exact_doc.get("response")

            new_vec = self._get_embedding(prompt)
            docs = self._load_db_documents()
            if not docs:
                return None
            best_score, best_payload = self._find_best_match(new_vec, docs)
            logger.debug("Cache similarity best_score=%s", best_score)
            if best_score >= self.threshold:
                return best_payload
            return None

        if not self.cache:
            return None

        new_vec = self._get_embedding(prompt)
        best_score = 0.0
        best_payload = None

        for _, (cached_vec, _, payload) in self.cache.items():
            if np is not None:
                denom = np.linalg.norm(new_vec) * np.linalg.norm(cached_vec)
                similarity = np.dot(new_vec, cached_vec) / denom if denom > 0 else 0.0
            else:
                denom = math.sqrt(sum(x * x for x in new_vec)) * math.sqrt(sum(x * x for x in cached_vec))
                similarity = sum(x * y for x, y in zip(new_vec, cached_vec)) / denom if denom > 0 else 0.0
            if similarity > best_score:
                best_score = similarity
                best_payload = payload

        logger.debug("Cache similarity best_score=%s", best_score)
        if best_score >= self.threshold:
            return best_payload
        return None

    def update_cache(self, prompt: str, response: Dict[str, Any]) -> None:
        vec = self._get_embedding(prompt)
        if self._mongo_collection is not None:
            prompt_hash = self._prompt_hash(prompt)
            model_name = response.get("model") or response.get("model_name")
            existing_doc = self._mongo_collection.find_one({"prompt_hash": prompt_hash})
            doc = {
                "prompt": prompt,
                "prompt_hash": prompt_hash,
                "embedding": vec.tolist(),
                "response": response,
                "model_name": model_name,
                "last_used": datetime.utcnow(),
                "updated_at": datetime.utcnow(),
            }
            if existing_doc is not None:
                self._mongo_collection.update_one(
                    {"_id": existing_doc["_id"]},
                    {"$set": doc},
                )
            else:
                self._mongo_collection.insert_one(doc)
            return

        self.cache[self.counter] = (vec, prompt, response)
        self.counter += 1
