import { createAnthropic } from "@ai-sdk/anthropic"
import { createOpenAI } from "@ai-sdk/openai"
import { createOpenAICompatible } from "@ai-sdk/openai-compatible"
import { createGoogleGenerativeAI } from "@ai-sdk/google"
import type { LanguageModel } from "ai"
import type { ProviderConfig } from "./config.js"

/**
 * Create a language model from provider config.
 */
export function createLanguageModel(config: ProviderConfig): LanguageModel {
  switch (config.type) {
    case "anthropic": {
      const anthropic = createAnthropic({
        apiKey: config.apiKey,
        baseURL: config.baseUrl ?? undefined,
      })
      return anthropic(config.model)
    }

    case "openai": {
      const openai = createOpenAI({
        apiKey: config.apiKey,
        baseURL: config.baseUrl ?? undefined,
      })
      return openai(config.model)
    }

    case "openai-compatible": {
      if (!config.baseUrl) {
        throw new Error("baseUrl is required for openai-compatible provider")
      }
      const compatible = createOpenAICompatible({
        name: "openai-compatible",
        apiKey: config.apiKey,
        baseURL: config.baseUrl,
      })
      return compatible(config.model)
    }

    case "google": {
      const google = createGoogleGenerativeAI({
        apiKey: config.apiKey,
        baseURL: config.baseUrl ?? undefined,
      })
      return google(config.model)
    }

    default: {
      const _exhaustive: never = config.type
      throw new Error(`Unknown provider type: ${_exhaustive}`)
    }
  }
}
