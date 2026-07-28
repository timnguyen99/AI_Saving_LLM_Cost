// Idea and Co-author: Tim Nguyen (timothynnguyen9@gmail.com)

import { CachePayload, SemanticCacheRouter } from "./router";
import { ComplexityClassifier } from "./classifier";

export interface ChatMessage {
  role: string;
  content: string;
}

export interface OpenAIClientLike {
  chat?: {
    completions?: {
      create: (args: { messages: ChatMessage[]; [key: string]: unknown }) => Promise<unknown>;
    };
  };
  ChatCompletion?: {
    create: (args: { messages: ChatMessage[]; [key: string]: unknown }) => Promise<unknown>;
  };
}

export class AICacherWrapper {
  client: OpenAIClientLike;
  router: SemanticCacheRouter;
  simpleModelName: string;
  complexModelName: string;
  classifier: ComplexityClassifier;

  constructor(
    client: OpenAIClientLike,
    router: SemanticCacheRouter,
    simpleModelName = "gpt-mini",
    complexModelName = "gpt-4",
    classifier = new ComplexityClassifier(),
  ) {
    this.client = client;
    this.router = router;
    this.simpleModelName = simpleModelName;
    this.complexModelName = complexModelName;
    this.classifier = classifier;
  }

  private extractUserPrompt(messages: ChatMessage[]): string {
    return messages.filter((m) => m.role === "user").map((m) => m.content).join(" ");
  }

  private async serializeResponse(response: unknown): Promise<CachePayload> {
    if (response && typeof response === "object") {
      return response as CachePayload;
    }
    return { raw: String(response) };
  }

  private chooseModel(messages: ChatMessage[], options: Record<string, unknown>): string | null {
    if (options.model || options.model_name) {
      return null;
    }

    const userPrompt = this.extractUserPrompt(messages);
    return this.classifier.isLocalTask(userPrompt) ? this.simpleModelName : this.complexModelName;
  }

  async createCompletion(messages: ChatMessage[], options: Record<string, unknown> = {}): Promise<CachePayload> {
    const userPrompt = this.extractUserPrompt(messages);
    const cachedResponse = await this.router.checkCache(userPrompt);
    if (cachedResponse) {
      return {
        ...cachedResponse,
        cached_by: "AI-Cacher-Local",
        latency_ms: 5,
      };
    }

    const selectedModel = this.chooseModel(messages, options);
    if (selectedModel) {
      options.model = selectedModel;
    }

    const start = performance.now();
    let cloudResponse: unknown;

    if (this.client.chat?.completions?.create) {
      cloudResponse = await this.client.chat.completions.create({ messages, ...options });
    } else if (this.client.ChatCompletion?.create) {
      cloudResponse = await this.client.ChatCompletion.create({ messages, ...options });
    } else {
      throw new Error("Unsupported OpenAI client interface");
    }

    const duration = performance.now() - start;
    const responseDict = await this.serializeResponse(cloudResponse);
    const payload = {
      ...responseDict,
      cached_by: "Cloud-Server",
      latency_ms: duration,
      selected_model: selectedModel || options.model,
    };

    await this.router.updateCache(userPrompt, payload);
    return payload;
  }
}
