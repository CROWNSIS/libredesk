# Help assistant retrieval

Knowledge-only assistants use a scoped retrieval and evidence pipeline:

1. Filter the index to the assistant's allowed help centres. Fuse semantic and title-weighted lexical ranks, remove duplicate passage content, and retain up to 12 candidates. Cosine scores remain unmodified and are not probabilities of correctness.
2. Preserve each candidate article's introduction and ask the completion model to choose one workflow matching the operation and audience. The server restricts the next stage to that source, so teacher and administrator instructions cannot be mixed.
3. Re-rank passages within the chosen article so the global top-k cannot hide its procedure. Ask which of these passages directly answer the question. The response is a bounded JSON list of passage numbers. Empty selections abstain; malformed, duplicate, out-of-range, or oversized selections fail closed. The server renders the selected documentation and its stored citations verbatim, never model-authored instructions or URLs.

Preview and live knowledge-only conversations use the same selection and rendering functions. Tool-enabled assistants retain their existing tool workflow. Published/AI-enabled/help-centre eligibility remains enforced before ranking.

The completion service must support OpenAI-compatible `response_format: json_schema` and accommodate the candidate context (up to 32 KB of passage text plus framing and the question). The Spark deployment uses a 12,288-token context. A 45-second selection deadline bounds inference; provider errors remain visible rather than falling back to an unrelated top match. The server restores document order within each selected source, including after an index reload.

## Verification

Run Go checks with the repository's supported Go toolchain and an isolated PostgreSQL database. The focused packages are `internal/ai`, `internal/aiagent`, and `internal/stringutil`.

`scripts/eval-ai-retrieval.py` is an opt-in evaluation of the deployed OLEDU corpus. It authenticates with `LIBREDESK_EVAL_USER` (default `System`) and `LIBREDESK_SYSTEM_USER_PASSWORD`, and calls only the assistant preview endpoint. It does not create conversations or send customer messages. Pass an unused report path with `--output`. It checks actual procedural phrases as well as citations, and includes unsupported, other-product, and adversarial questions.

`TestEvaluateRetrievalCorpus` optionally exports actual hybrid candidates from a supplied cosine-ranked public corpus. Set `LIBREDESK_RETRIEVAL_CORPUS_INPUT` and `LIBREDESK_RETRIEVAL_CORPUS_OUTPUT` to opt in. This is an experiment harness; the independent regression tests establish authorization and ranking invariants.

## Audit baseline

The original deployed assistant passed 3 of 15 procedural/abstention checks on 2026-09-04. It returned a generic support article for lesson creation, often selected introductory or safety paragraphs instead of steps, and accepted several unsupported requests. Merely changing the completion model could not repair it because the original knowledge-only path bypassed that model entirely.

Qwen embedding query instructions were tested against the live corpus. They improved some topic matches but did not resolve the original procedure-selection failure or related-software false positives. The existing embedding model and vectors were therefore retained for this change.

The evaluation is a regression sample, not a guarantee of answer accuracy on every question. The single-source design deliberately favors one coherent workflow over synthesizing multiple articles; complex cross-article questions may abstain or need a follow-up. Never treat model selection as a replacement for help-centre authorization.
