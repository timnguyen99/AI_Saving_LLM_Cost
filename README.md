Idea and Co-author: Tim Nguyen (timothynnguyen9@gmail.com)

# AI Local Middleware Packages

This repository contains three language-specific AI caching middleware packages.

- `ai_cacher` — Python semantic cache + cloud interceptor
- `ai_cacher_react` — React/TypeScript semantic cache wrapper
- `ai_cacher_angular` — Angular/TypeScript semantic cache wrapper
- `ai_cacher_go` — Go semantic cache wrapper

## Core idea

Each package provides a local semantic cache with:
- prompt similarity matching
- local cache hit return
- cloud forwarding on cache miss
- cache update for future reuse

The Python package also supports optional MongoDB persistence and simple-vs-complex model selection.

---

## Python Usage (`ai_cacher`)

### Setup

```bash
cd /Users/trnguyen/Development/Apps/AILocal
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

### Basic example

```python
from ai_cacher import SemanticCacheRouter, AICacherWrapper
from openai import OpenAI

router = SemanticCacheRouter(threshold=0.95)
real_client = OpenAI()
wrapper = AICacherWrapper(real_client, router)

messages = [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Summarize the latest quarterly report."},
]

response = wrapper.create_completion(messages=messages, model="gpt-4o")
print(response)
```

### MongoDB-backed cache (optional)

If you want persistence across process restarts, pass `mongo_uri` when creating the router.

```python
router = SemanticCacheRouter(
    threshold=0.95,
    mongo_uri="mongodb://localhost:27017",
    mongo_db_name="ai_cacher",
    mongo_collection_name="prompt_cache",
)
```

This stores:
- prompt text
- prompt hash
- prompt embedding
- cloud response payload
- last used timestamp

### Simple vs complex model routing

The wrapper can choose a smaller model for simple tasks and a larger model for harder tasks:

```python
wrapper = AICacherWrapper(
    real_client,
    router,
    simple_model_name="gpt-mini",
    complex_model_name="gpt-4",
)
```

If `model` or `model_name` is already passed into `create_completion`, it is preserved.

### Key behavior

- `SemanticCacheRouter.check_cache(prompt)` returns a cached response if similarity is above threshold.
- `AICacherWrapper.create_completion(...)` returns cached responses instantly when possible.
- On cache miss, it calls the real client and stores the response.
- Cached payloads include `cached_by` and `latency_ms`.

### Example with direct wrapper call

```python
response = wrapper.create_completion(
    messages=messages,
    model="gpt-4o",
    temperature=0.7,
)
```

### Notes

- If `sentence-transformers` is installed, local embeddings are generated with `all-MiniLM-L6-v2`.
- If not installed, the router uses a token-frequency fallback embedding.
- Use `threshold` to tune how similar prompts must be before reuse.

---

## React / TypeScript Usage (`ai_cacher_react`)

### Setup

```bash
cd /Users/trnguyen/Development/Apps/AILocal/ai_cacher_react
npm install
```

### Example

```ts
import { SemanticCacheRouter, AICacherWrapper } from "./src";

const router = new SemanticCacheRouter(0.95);
const wrapper = new AICacherWrapper(openaiClient, router);

const messages = [
  { role: "system", content: "You are a helpful assistant." },
  { role: "user", content: "Rewrite this paragraph for clarity." },
];

const response = await wrapper.createCompletion(messages, { model: "gpt-4o" });
console.log(response);
```

### Notes

- Works in browser and Node environments.
- Uses lightweight prompt matching for cache reuse.
- Cache hit responses include `latency_ms`.

---

## Angular / TypeScript Usage (`ai_cacher_angular`)

### Setup

```bash
cd /Users/trnguyen/Development/Apps/AILocal/ai_cacher_angular
npm install
```

### Example

```ts
import { SemanticCacheRouter, AICacherWrapper } from "./src";

const router = new SemanticCacheRouter(0.95);
const wrapper = new AICacherWrapper(openaiClient, router);

const messages = [
  { role: "system", content: "You are a helpful assistant." },
  { role: "user", content: "Summarize the product benefits." },
];

const response = await wrapper.createCompletion(messages, { model: "gpt-4o" });
console.log(response);
```

### Notes

- Designed for Angular/TypeScript projects with the same API as the React package.
- Can be used with browser or server-side rendering setups.
- Reuses cached semantic responses and preserves `latency_ms` metadata.

---

## Go Usage (`ai_cacher_go`)

### Setup

```bash
cd /Users/trnguyen/Development/Apps/AILocal/ai_cacher_go
go mod tidy
```

### Example

```go
package main

import (
    "fmt"
    "ai_cacher_go"
)

func main() {
    router := ai_cacher_go.NewSemanticCacheRouter(0.95)
    client := NewOpenAIClient() // implement OpenAIClient
    wrapper := ai_cacher_go.NewAICacherWrapper(client, router)

    messages := []ai_cacher_go.ChatMessage{
        {Role: "system", Content: "You are a helpful assistant."},
        {Role: "user", Content: "Generate a short product summary."},
    }

    response, err := wrapper.CreateCompletion(messages, map[string]any{"model": "gpt-4o"})
    if err != nil {
        panic(err)
    }

    fmt.Printf("Response: %+v\n", response)
}
```

### Notes

- Implement `OpenAIClient` or similar interface for your real OpenAI client.
- The Go router uses local semantic matching and caches cloud responses.

---

## Demo: compare cached vs direct cloud usage

A Python demo is included to show token reduction and cache-hit behavior.

### Run the demo

```bash
cd /Users/trnguyen/Development/Apps/AILocal
python3 demo_cost_comparison.py
```

### What it demonstrates

- direct cloud requests to a mock client
- the same requests through `ai_cacher`
- token usage with and without caching
- cache hits shown by `cached_by = "AI-Cacher-Local"`

### Expected behavior

When prompts repeat or are semantically similar:
- fewer cloud calls are made
- cached responses are reused
- total token usage decreases

---

## Project layout

- `ai_cacher/` — Python package
- `ai_cacher_react/` — React/TypeScript package
- `ai_cacher_go/` — Go package
- `demo_cost_comparison.py` — Python cache efficiency demo
- `ailocal.md` — design notes and architecture details

## Innovator

- Tim Nguyen

---

## License

Use as needed for prototype or internal experimentation.
