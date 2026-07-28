// Idea and Co-author: Tim Nguyen (timothynnguyen9@gmail.com)

import { AICacherWrapper } from "./client";
import { SemanticCacheRouter } from "./router";
import { ChatMessage } from "./client";

class MockOpenAIClient {
  chat = {
    completions: {
      create: async ({ messages }: { messages: ChatMessage[] }) => {
        const prompt = messages
          .filter((m) => m.role === "user")
          .map((m) => m.content)
          .join(" ");
        const responseText = `AI response for prompt: ${prompt.slice(0, 120)}`;
        const promptTokens = Math.max(1, prompt.split(/\s+/).filter(Boolean).length);
        const completionTokens = Math.max(1, responseText.split(/\s+/).filter(Boolean).length);
        return {
          choices: [
            {
              message: {
                role: "assistant",
                content: responseText,
              },
            },
          ],
          usage: {
            prompt_tokens: promptTokens,
            completion_tokens: completionTokens,
          },
        };
      },
    },
  };
}

type DemoResult = {
  cached_by?: string;
  latency_ms?: number;
  usage?: {
    prompt_tokens?: number;
    completion_tokens?: number;
  };
  [key: string]: unknown;
};

function countTokens(payload: DemoResult): number {
  const usage = payload.usage;
  return (usage?.prompt_tokens ?? 0) + (usage?.completion_tokens ?? 0);
}

async function runDirect(client: MockOpenAIClient, messages: ChatMessage[]): Promise<DemoResult> {
  const response = await client.chat.completions.create({ messages });
  return {
    ...response,
    cached_by: "Direct-Cloud",
    latency_ms: 1500,
  };
}

async function runCached(wrapper: AICacherWrapper, messages: ChatMessage[]): Promise<DemoResult> {
  return wrapper.createCompletion(messages);
}

async function main(): Promise<void> {
  const messagesBatch: ChatMessage[][] = [
    [
      { role: "system", content: "You are a helpful assistant." },
      { role: "user", content: "Summarize the monthly sales report." },
    ],
    [
      { role: "system", content: "You are a helpful assistant." },
      { role: "user", content: "Summarize the monthly sales report." },
    ],
    [
      { role: "system", content: "You are a helpful assistant." },
      { role: "user", content: "Convert this list to JSON: apple, banana, cherry." },
    ],
    [
      { role: "system", content: "You are a helpful assistant." },
      { role: "user", content: "Summarize the monthly sales report." },
    ],
    [
      { role: "system", content: "You are a helpful assistant." },
      { role: "user", content: "Convert this list to JSON: apple, banana, cherry." },
    ],
  ];

  const mockClient = new MockOpenAIClient();
  const router = new SemanticCacheRouter(0.88);
  const wrapper = new AICacherWrapper(mockClient, router);

  let directTokens = 0;
  let cachedCloudTokens = 0;
  let cachedHits = 0;
  let directCalls = 0;
  let cachedCalls = 0;

  console.log("=== Direct cloud usage ===");
  for (const messages of messagesBatch) {
    const result = await runDirect(mockClient, messages);
    directCalls += 1;
    const tokens = countTokens(result);
    directTokens += tokens;
    console.log(`Call ${directCalls}: prompt='${messages[messages.length - 1].content}' tokens=${tokens}`);
  }

  console.log("\n=== Cached usage ===");
  for (const messages of messagesBatch) {
    const result = await runCached(wrapper, messages);
    if (result.cached_by === "AI-Cacher-Local") {
      cachedHits += 1;
    } else {
      cachedCalls += 1;
      cachedCloudTokens += countTokens(result);
    }
    console.log(
      `Request: '${messages[messages.length - 1].content}' cached_by=${result.cached_by} tokens=${
        result.cached_by === "AI-Cacher-Local" ? 0 : countTokens(result)
      }`,
    );
  }

  console.log("\n=== Summary ===");
  console.log(`Direct cloud calls: ${directCalls}`);
  console.log(`Cached cloud calls: ${cachedCalls}`);
  console.log(`Local cache hits: ${cachedHits}`);
  console.log(`Total direct tokens: ${directTokens}`);
  console.log(`Total cached cloud tokens: ${cachedCloudTokens}`);
  console.log(`Token reduction: ${directTokens - cachedCloudTokens} tokens`);
  console.log(
    `Estimated cost savings: $${((directTokens - cachedCloudTokens) * 0.000002).toFixed(6)} (assuming $0.002 per 1,000 tokens)`,
  );
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
