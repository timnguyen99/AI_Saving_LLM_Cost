// Idea and Co-author: Tim Nguyen (timothynnguyen9@gmail.com)

export class ComplexityClassifier {
  private static LOCAL_KEYWORDS = [
    "format",
    "reformat",
    "json",
    "csv",
    "yaml",
    "markdown",
    "summarize",
    "extract",
    "clean",
    "normalize",
    "validate",
    "transform",
    "tokenize",
  ];

  private static CLOUD_KEYWORDS = [
    "javascript",
    "python",
    "sql",
    "code",
    "complex",
    "reason",
    "analysis",
    "multi-step",
    "strategy",
    "plan",
    "problem",
    "math",
    "proof",
    "logic",
  ];

  isLocalTask(userText: string, systemPrompt?: string): boolean {
    const combined = [systemPrompt ?? "", userText ?? ""].join(" ").toLowerCase();
    const localScore = ComplexityClassifier.LOCAL_KEYWORDS.reduce(
      (sum, term) => sum + (combined.includes(term) ? 1 : 0),
      0,
    );
    const cloudScore = ComplexityClassifier.CLOUD_KEYWORDS.reduce(
      (sum, term) => sum + (combined.includes(term) ? 1 : 0),
      0,
    );

    if (localScore >= 2 && cloudScore === 0) {
      return true;
    }
    if (cloudScore >= 2 && localScore === 0) {
      return false;
    }
    if (combined.includes("simple") || combined.includes("basic")) {
      return true;
    }

    return localScore >= cloudScore;
  }
}
