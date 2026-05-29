package cmd

// defaultLocalModel is the default Ollama model for musu-marketer reasoning.
// Per operator direction the local stack uses TurboQuant Gemma 4 — Google's
// Gemma 4 (E4B-class) with int4 + KV-cache quantization (TurboQuant Stage 1),
// efficient on the 16GB box yet far stronger than the old Gemma 3 4B at strict
// citation. Override any time with --model. Pull it with:
//
//	ollama pull ssfdre38/gemma4-turbo
//
// musu's "0 external LLM default" invariant holds: this is a local endpoint.
const defaultLocalModel = "ssfdre38/gemma4-turbo"
