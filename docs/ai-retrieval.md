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

## Deployed result: 2026-09-04

- Runtime revision: `daa61c0b3afcf9b23f9d8f7cbfb07a99436d4d7e`; image digest `sha256:4508d4ad23f8b722c40ee169069d0957f61a0716e74d0d78fef0ea878847102a`. [Release CI](https://github.com/CROWNSIS/libredesk/actions/runs/33918912768) passed tests, build, and publication. Deployment completed with the existing backup/health-check procedure.
- Completion: `qwen3:14b` on Spark through Ollama 0.32.15, temperature 0, reasoning disabled, 160 output tokens, 12,288-token context. Native provider tests passed; the model reports 100% GPU placement (approximately 11 GB).
- Embeddings remain `qwen3-embedding:0.6b`, 1024 dimensions, on the separate Spark embedding service. All 102 published AI-enabled articles retained nonempty index fingerprints.
- The original 15-question set improved from 3/15 to 15/15. Six additional questions cover rephrasing, teacher versus administrator grading, another product, and instruction injection. All 21 passed on each of three consecutive live-preview rounds: **63/63 checks**.
- Median latency across all checks: 1.94 seconds. Across the 39 supported-answer checks: 2.36 seconds median, 7.76 seconds maximum. These are sequential preview observations, not a concurrency benchmark; other Spark workloads can affect latency.
- Final reports are local, untracked artifacts under `output/ai-audit-20260904-14b-round{1,2,3}.json` and `output/ai-audit-20260904-14b-additional-round{1,2,3}.json`. They contain synthetic questions and public help-guide answers, not credentials or customer conversations.

Earlier 8B releases improved retrieval but failed a teacher phrasing or occasionally abstained on the original teacher question. Source-level routing, article-specific evidence expansion, and the 14B model were all retained in the final configuration. No special-case article IDs or fixed answers were added to production code. No customer messages were sent during evaluation.
