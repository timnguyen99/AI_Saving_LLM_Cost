// Idea and Co-author: Tim Nguyen (timothynnguyen9@gmail.com)

import { MongoClient, Collection } from "mongodb";

export type CachePayload = Record<string, unknown>;

export interface CacheDocument {
  prompt: string;
  promptHash: string;
  vector: number[];
  payload: CachePayload;
  updatedAt: Date;
}

export interface CacheStoreAdapter {
  findByPromptHash(promptHash: string): Promise<CacheDocument | null>;
  findAll(): Promise<CacheDocument[]>;
  upsert(doc: CacheDocument): Promise<void>;
}

function hashPrompt(prompt: string): string {
  let hash = 5381;
  for (let i = 0; i < prompt.length; i += 1) {
    hash = (hash << 5) + hash + prompt.charCodeAt(i);
    hash &= 0xffffffff;
  }
  return (hash >>> 0).toString(16);
}

export class SemanticCacheRouter {
  threshold: number;
  cache: Map<number, CacheDocument>;
  counter: number;
  store?: CacheStoreAdapter;

  constructor(threshold = 0.95, store?: CacheStoreAdapter) {
    this.threshold = threshold;
    this.cache = new Map();
    this.counter = 0;
    this.store = store;
  }

  private _getEmbedding(text: string): number[] {
    const tokens = text
      .toLowerCase()
      .split(/\s+/)
      .filter(Boolean);

    const freq: Record<string, number> = {};
    tokens.forEach((token) => {
      freq[token] = (freq[token] || 0) + 1;
    });

    const vector = new Array<number>(128).fill(0);
    const keys = Object.keys(freq).sort();
    keys.slice(0, 128).forEach((token, index) => {
      vector[index] = freq[token];
    });

    const length = Math.sqrt(vector.reduce((sum, value) => sum + value * value, 0));
    return length > 0 ? vector.map((value) => value / length) : vector;
  }

  private async _findBestMatch(newVec: number[], docs: CacheDocument[]): Promise<{ score: number; payload: CachePayload | null }> {
    let bestScore = 0;
    let bestPayload: CachePayload | null = null;

    docs.forEach((doc) => {
      const dot = newVec.reduce((sum, value, idx) => sum + value * (doc.vector[idx] ?? 0), 0);
      if (dot > bestScore) {
        bestScore = dot;
        bestPayload = doc.payload;
      }
    });

    return { score: bestScore, payload: bestPayload };
  }

  async checkCache(prompt: string): Promise<CachePayload | null> {
    if (this.store) {
      const promptHash = hashPrompt(prompt);
      const exactDoc = await this.store.findByPromptHash(promptHash);
      if (exactDoc) {
        return exactDoc.payload;
      }

      const docs = await this.store.findAll();
      if (docs.length === 0) {
        return null;
      }

      const { score, payload } = await this._findBestMatch(this._getEmbedding(prompt), docs);
      return score >= this.threshold ? payload : null;
    }

    if (this.cache.size === 0) {
      return null;
    }

    const newVec = this._getEmbedding(prompt);
    let bestScore = 0;
    let bestPayload: CachePayload | null = null;

    Array.from(this.cache.values()).forEach((entry) => {
      const dot = newVec.reduce((sum, value, idx) => sum + value * (entry.vector[idx] ?? 0), 0);
      if (dot > bestScore) {
        bestScore = dot;
        bestPayload = entry.payload;
      }
    });

    return bestScore >= this.threshold ? bestPayload : null;
  }

  async updateCache(prompt: string, response: CachePayload): Promise<void> {
    const entry: CacheDocument = {
      prompt,
      promptHash: hashPrompt(prompt),
      vector: this._getEmbedding(prompt),
      payload: response,
      updatedAt: new Date(),
    };

    if (this.store) {
      await this.store.upsert(entry);
      return;
    }

    this.cache.set(this.counter, entry);
    this.counter += 1;
  }
}

export class MongoCacheStore implements CacheStoreAdapter {
  private collection: Collection<CacheDocument>;

  private constructor(collection: Collection<CacheDocument>) {
    this.collection = collection;
  }

  static async connect(
    uri: string,
    dbName = "ai_cacher",
    collectionName = "prompt_cache",
  ): Promise<MongoCacheStore> {
    const client = new MongoClient(uri);
    await client.connect();
    const collection = client.db(dbName).collection<CacheDocument>(collectionName);
    await collection.createIndex({ promptHash: 1 }, { unique: true });
    return new MongoCacheStore(collection);
  }

  async findByPromptHash(promptHash: string): Promise<CacheDocument | null> {
    return this.collection.findOne({ promptHash });
  }

  async findAll(): Promise<CacheDocument[]> {
    return this.collection.find().toArray();
  }

  async upsert(doc: CacheDocument): Promise<void> {
    await this.collection.updateOne(
      { promptHash: doc.promptHash },
      { $set: doc },
      { upsert: true },
    );
  }
}
