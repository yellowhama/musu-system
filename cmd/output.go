package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/viper"
)

type jsonResponse struct {
	Status        string      `json:"status"`
	Message       string      `json:"message,omitempty"`
	Data          interface{} `json:"data,omitempty"`
	ActionableFix string      `json:"actionable_fix,omitempty"`
}

func printJSONSuccess(message string, data interface{}) {
	if !viper.GetBool("json") {
		return
	}
	encoded, _ := json.MarshalIndent(jsonResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	}, "", "  ")
	fmt.Println(string(encoded))
}

func printJSONError(err error, data interface{}, actionableFix string) {
	if !viper.GetBool("json") {
		return
	}
	encoded, _ := json.MarshalIndent(jsonResponse{
		Status:        "error",
		Message:       err.Error(),
		Data:          data,
		ActionableFix: actionableFix,
	}, "", "  ")
	fmt.Println(string(encoded))
}
