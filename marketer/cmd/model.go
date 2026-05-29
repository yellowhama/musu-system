package cmd

// defaultLocalModel is the default Ollama model for musu-marketer reasoning.
// Per operator direction the local stack uses a TurboQuant Gemma (Gemma 3 4B
// class) — small enough to run on the 16GB box yet strong on Korean. Override
// any time with --model. Confirm the exact pulled tag matches this string:
//
//	ollama pull gemma3:4b      # then: --model gemma3:4b
//
// musu's "0 external LLM default" invariant holds: this is a local endpoint.
const defaultLocalModel = "gemma3:4b"
