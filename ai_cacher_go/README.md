# AI Cacher Go

Go package for semantic caching and OpenAI-like request interception.

## Usage

1. Initialize:

   ```go
   router := ai_cacher_go.NewSemanticCacheRouter(0.88)
   wrapper := ai_cacher_go.NewAICacherWrapper(client, router)
   ```

2. Use wrapper:

   ```go
   resp, err := wrapper.CreateCompletion(messages, params)
   ```
