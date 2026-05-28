package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// JSONResponse defines the standard machine-readable output format.
type JSONResponse struct {
	Status        string      `json:"status"`
	Message       string      `json:"message,omitempty"`
	Data          interface{} `json:"data,omitempty"`
	ActionableFix string      `json:"actionable_fix,omitempty"`
}

// PrintInfo prints informational messages to stderr (to keep stdout clean for JSON).
func PrintInfo(format string, a ...interface{}) {
	if viper.GetBool("json") {
		// Silent in JSON mode or print to stderr
		return
	}
	fmt.Printf(format+"\n", a...)
}

// PrintSuccess prints success messages with an emoji if not in JSON mode.
func PrintSuccess(msg string, a ...interface{}) {
	if viper.GetBool("json") {
		return
	}
	fmt.Printf("✅ "+msg+"\n", a...)
}

// PrintError prints error messages.
func PrintError(err error, fix string) {
	if viper.GetBool("json") {
		PrintJSONError(err.Error(), nil, fix)
		return
	}
	fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
	if fix != "" {
		fmt.Fprintf(os.Stderr, "💡 Tip: %s\n", fix)
	}
}

// PrintJSON prints a final successful result in JSON format and exits.
func PrintJSON(message string, data interface{}) {
	if !viper.GetBool("json") {
		return
	}
	resp := JSONResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	}
	encoded, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(encoded))
}

func PrintJSONError(message string, data interface{}, actionableFix string) {
	if !viper.GetBool("json") {
		return
	}
	resp := JSONResponse{
		Status:        "error",
		Message:       message,
		Data:          data,
		ActionableFix: actionableFix,
	}
	encoded, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(encoded))
}
